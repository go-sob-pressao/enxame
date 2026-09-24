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
	"github.com/go-sob-pressao/enxame/internal/worker/runner"
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
	ReportEvery    time.Duration
	Worker         string
	Now            func() time.Time
	Log            *slog.Logger

	executados atomic.Int64
}

// ErrAttemptTimeout é a causa registrada quando a tentativa estoura o
// prazo.
var ErrAttemptTimeout = errors.New("tentativa excedeu o prazo")

// livro:inicio pool-m1

// Run executa até o contexto ser cancelado ou a fila falhar. Um poller
// busca; Concurrency workers executam; o errgroup propaga o primeiro
// erro e cancela os demais.
func (p *Pool) Run(ctx context.Context) error {
	jobs := make(chan job.Job)
	g, ctx := errgroup.WithContext(ctx)

	g.Go(func() error { return p.buscar(ctx, jobs) })
	for range p.Concurrency {
		g.Go(func() error { return p.trabalhar(ctx, jobs) })
	}
	go p.relatar()

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
	resultado := make(chan error)
	go func() {
		resultado <- runner.Call(ctx, h, j)
	}()

	var err error
	select {
	case err = <-resultado:
	case <-time.After(p.AttemptTimeout):
		err = ErrAttemptTimeout
	}
	p.executados.Add(1)
	if err != nil {
		agora := p.Now()
		return p.Queue.Fail(
			j.ID,
			agora,
			err.Error(),
			errors.Is(err, runner.ErrPermanent),
			agora.Add(p.RetryDelay),
		)
	}
	return p.Queue.Complete(j.ID, p.Now())
}

// relatar registra o progresso periodicamente.
func (p *Pool) relatar() {
	for range time.NewTicker(p.ReportEvery).C {
		p.Log.Info(
			"progresso",
			slog.String("fila", p.QueueName),
			slog.Int64("executados", p.executados.Load()),
		)
	}
}

// livro:fim pool-m1
