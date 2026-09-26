package worker

import (
	"context"
	"errors"
	"log/slog"
	"sync/atomic"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/go-sob-pressao/enxame/internal/core/id"
	"github.com/go-sob-pressao/enxame/internal/core/job"
	"github.com/go-sob-pressao/enxame/internal/queue"
	"github.com/go-sob-pressao/enxame/internal/worker/heartbeat"
	"github.com/go-sob-pressao/enxame/internal/worker/runner"
	pkgjob "github.com/go-sob-pressao/enxame/pkg/job"
)

// Queue é o que o pool precisa da fila, declarado aqui, no consumidor.
type Queue interface {
	queue.Source
	queue.Notifier
	Complete(jid id.JobID, at time.Time) error
	Fail(
		jid id.JobID,
		at time.Time,
		cause string,
		permanent bool,
		retryAt time.Time,
	) error
	Promote(at time.Time) (int, error)
}

// Pool executa os jobs de uma fila com concorrência limitada.
type Pool struct {
	Queue          Queue
	QueueName      string
	Handlers       map[string]runner.Handler
	Concurrency    int
	PollTimeout    time.Duration
	AttemptTimeout time.Duration
	RetryDelay     time.Duration
	// Backoff, se definido, substitui RetryDelay: recebe o número da
	// tentativa que falhou e devolve a espera até a próxima.
	Backoff func(attempt int) time.Duration
	// Heartbeat, se definido, é chamado a cada HeartbeatEvery enquanto
	// o handler roda. Um batimento recusado cancela a tentativa.
	Heartbeat      func(context.Context, id.JobID, int) error
	HeartbeatEvery time.Duration
	ReportEvery    time.Duration
	Worker         string
	Now            func() time.Time
	Log            *slog.Logger

	executados, falhas, panicos atomic.Int64
	rodando                     atomic.Bool
}

// livro:inicio stats-corrigido

// Stats são os contadores do pool, para métricas e para o painel.
type Stats struct {
	Executed int64 // tentativas concluídas, com sucesso ou não
	Failed   int64 // tentativas que falharam
	Panics   int64 // tentativas que entraram em pânico
}

// Stats devolve um retrato dos contadores. Cada contador é lido de
// forma atômica; o retrato como um todo não é — Executed pode incluir
// uma tentativa cuja falha ainda não entrou em Failed. Para um painel,
// basta.
func (p *Pool) Stats() Stats {
	return Stats{
		Executed: p.executados.Load(),
		Failed:   p.falhas.Load(),
		Panics:   p.panicos.Load(),
	}
}

// Running informa se o pool está executando — a probe de saúde usa.
func (p *Pool) Running() bool { return p.rodando.Load() }

// livro:fim stats-corrigido

// ErrAttemptTimeout é a causa registrada quando a tentativa estoura o
// prazo.
var ErrAttemptTimeout = errors.New("tentativa excedeu o prazo")

// livro:inicio pool-m1-corrigido

// Run executa até o contexto ser cancelado ou a fila falhar. Um poller
// busca; Concurrency workers executam; o errgroup propaga o primeiro
// erro e cancela os demais.
func (p *Pool) Run(ctx context.Context) error {
	p.rodando.Store(true)
	defer p.rodando.Store(false)
	jobs := make(chan job.Job)
	g, ctx := errgroup.WithContext(ctx)

	g.Go(func() error { return p.buscar(ctx, jobs) })
	for range p.Concurrency {
		g.Go(func() error { return p.trabalhar(ctx, jobs) })
	}
	g.Go(func() error { return p.relatar(ctx) })

	if err := g.Wait(); !errors.Is(err, context.Canceled) {
		return err
	}
	return nil
}

func (p *Pool) buscar(ctx context.Context, jobs chan<- job.Job) error {
	cfg := queue.PollConfig{
		Queue:       p.QueueName,
		Worker:      p.Worker,
		PollTimeout: p.PollTimeout,
		Now:         p.Now,
	}
	for {
		if _, err := p.Queue.Promote(p.Now()); err != nil {
			return err
		}
		j, ok, err := queue.Poll(ctx, p.Queue, p.Queue, cfg)
		if err != nil {
			return err
		}
		if !ok {
			continue
		}
		select {
		case jobs <- j:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func (p *Pool) trabalhar(
	ctx context.Context,
	jobs <-chan job.Job,
) error {
	for {
		select {
		case j := <-jobs:
			if err := p.executar(ctx, j); err != nil {
				return err
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

// executar roda uma tentativa com prazo. Se o handler não terminar a
// tempo, a tentativa é registrada como falha e o worker segue adiante.
//
// Dono e término da goroutine do handler: o canal tem buffer de 1,
// então o envio nunca bloqueia; e o contexto da tentativa é cancelado
// no prazo, então o handler que respeita contexto termina logo depois.
// O handler que ignora o contexto continua vivo até terminar — e
// aparece no perfil goroutineleak.
func (p *Pool) executar(ctx context.Context, j job.Job) error {
	h, ok := p.Handlers[j.Kind]
	if !ok {
		return p.Queue.Fail(
			j.ID,
			p.Now(),
			"kind sem handler: "+j.Kind,
			true,
			time.Time{},
		)
	}
	tentativa, cancelar := context.WithTimeoutCause(
		ctx,
		p.AttemptTimeout,
		ErrAttemptTimeout,
	)
	defer cancelar()
	if p.Heartbeat != nil && p.HeartbeatEvery > 0 {
		var perder context.CancelCauseFunc
		tentativa, perder = context.WithCancelCause(tentativa)
		defer perder(nil)
		go heartbeat.Run(tentativa, p.HeartbeatEvery,
			func(ctx context.Context) error {
				return p.Heartbeat(ctx, j.ID, j.Attempt)
			}, perder)
	}
	tentativa = pkgjob.WithInfo(tentativa, pkgjob.Info{
		ID: j.ID.String(), Kind: j.Kind, Attempt: j.Attempt,
		MaxAttempts:    j.MaxAttempts,
		IdempotencyKey: id.JobKey(j.Namespace, j.ID),
	})
	resultado := make(chan error, 1)
	go func() {
		resultado <- runner.Call(tentativa, h, j)
	}()

	var err error
	select {
	case err = <-resultado:
	case <-tentativa.Done():
		err = context.Cause(tentativa)
	}
	p.executados.Add(1)
	if errors.Is(err, heartbeat.ErrPerdida) {
		return p.perdida(j, err)
	}
	if err != nil {
		p.falhas.Add(1)
		if _, ok := errors.AsType[*runner.PanicError](err); ok {
			p.panicos.Add(1)
		}
		agora := p.Now()
		return p.registrar(j, p.Queue.Fail(
			j.ID,
			agora,
			err.Error(),
			errors.Is(err, runner.ErrPermanent),
			agora.Add(p.espera(j.Attempt)),
		))
	}
	return p.registrar(j, p.Queue.Complete(j.ID, p.Now()))
}

// livro:inicio perdida

// registrar trata a recusa do domínio ao registrar o fim da tentativa:
// o job foi resgatado enquanto esta tentativa rodava, e já pertence a
// outra. Não é um erro do pool — é o resgate funcionando —, e o pool
// segue com os outros jobs.
func (p *Pool) registrar(j job.Job, err error) error {
	if errors.Is(err, job.ErrInvalidTransition) {
		return p.perdida(j, err)
	}
	return err
}

func (p *Pool) perdida(j job.Job, err error) error {
	p.Log.Warn("tentativa perdida",
		slog.String("job", j.ID.String()),
		slog.Int("tentativa", j.Attempt), slog.Any("erro", err))
	return nil
}

// livro:fim perdida

// relatar registra o progresso periodicamente, até o contexto terminar.
func (p *Pool) relatar(ctx context.Context) error {
	t := time.NewTicker(p.ReportEvery)
	defer t.Stop()
	for {
		select {
		case <-t.C:
			p.Log.InfoContext(
				ctx,
				"progresso",
				slog.String("fila", p.QueueName),
				slog.Int64("executados", p.executados.Load()),
			)
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

// livro:fim pool-m1-corrigido

// espera devolve quanto esperar antes da próxima tentativa.
func (p *Pool) espera(attempt int) time.Duration {
	if p.Backoff != nil {
		return p.Backoff(attempt)
	}
	return p.RetryDelay
}
