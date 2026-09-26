package enxame

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"math/rand/v2"
	"os"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/go-sob-pressao/enxame/internal/core/job"
	"github.com/go-sob-pressao/enxame/internal/core/policy"
	"github.com/go-sob-pressao/enxame/internal/core/schedule"
	"github.com/go-sob-pressao/enxame/internal/delivery"
	"github.com/go-sob-pressao/enxame/internal/store/postgres"
	"github.com/go-sob-pressao/enxame/internal/worker"
	"github.com/go-sob-pressao/enxame/internal/worker/runner"
	wf "github.com/go-sob-pressao/enxame/internal/worker/workflow"
	"github.com/go-sob-pressao/enxame/pkg/webhook"
	"github.com/go-sob-pressao/enxame/pkg/workflow"
)

// Job é o job como o handler o vê.
type Job struct {
	ID          string
	Kind        string
	Args        json.RawMessage
	Attempt     int
	MaxAttempts int
}

// Handler processa um job. Um erro devolve o job à fila, para nova
// tentativa com backoff; Permanent(err) o descarta.
type Handler func(ctx context.Context, j Job) error

// Permanent marca um erro que não adianta repetir.
func Permanent(err error) error { return runner.Permanent(err) }

// Workflow é uma função de workflow registrada.
type Workflow func(
	c *workflow.Context,
	input json.RawMessage,
) (any, error)

// WorkerConfig configura o worker; os zeros têm padrão.
type WorkerConfig struct {
	Queues      map[string]int // fila → workers; {"default": 10}
	RetryBase   time.Duration  // 1 s
	RetryMax    time.Duration  // 10 min
	RescueAfter time.Duration  // 5 min: prazo de uma tentativa órfã
	Name        string         // hostname-pid
	Log         *slog.Logger
	// O breaker da entrega de webhooks, por endpoint: quantas falhas
	// seguidas o abrem (5) e por quanto tempo fica aberto (1 min).
	BreakerLimiar int
	BreakerPausa  time.Duration
}

// Worker executa jobs, avança workflows, dispara agendamentos e resgata
// tentativas órfãs, no processo da aplicação.
type Worker struct {
	c         *Client
	cfg       WorkerConfig
	handlers  map[string]runner.Handler
	workflows map[string]wf.Func
}

// NewWorker cria um worker sobre o banco do cliente.
func (c *Client) NewWorker(cfg WorkerConfig) *Worker {
	if len(cfg.Queues) == 0 {
		cfg.Queues = map[string]int{"default": 10}
	}
	cfg.RetryBase = cmp(cfg.RetryBase, time.Second)
	cfg.RetryMax = cmp(cfg.RetryMax, 10*time.Minute)
	cfg.RescueAfter = cmp(cfg.RescueAfter, 5*time.Minute)
	if cfg.Name == "" {
		h, _ := os.Hostname()
		cfg.Name = fmt.Sprintf("%s-%d", h, os.Getpid())
	}
	if cfg.Log == nil {
		cfg.Log = slog.New(slog.DiscardHandler)
	}
	return &Worker{c: c, cfg: cfg,
		handlers:  map[string]runner.Handler{},
		workflows: map[string]wf.Func{}}
}

func cmp(v, padrao time.Duration) time.Duration {
	if v <= 0 {
		return padrao
	}
	return v
}

// Handle registra o handler de um kind.
func (w *Worker) Handle(kind string, h Handler) {
	w.handlers[kind] = func(ctx context.Context, j job.Job) error {
		return h(ctx, Job{ID: j.ID.String(), Kind: j.Kind,
			Args: j.Args, Attempt: j.Attempt,
			MaxAttempts: j.MaxAttempts})
	}
}

// Workflow registra uma função de workflow.
func (w *Worker) Workflow(tipo string, fn Workflow) {
	w.workflows[tipo] = wf.Func(fn)
}

// livro:inicio worker

// Run executa até o contexto terminar ou uma peça falhar. É o pool da
// Parte I, na terceira e última forma: um por fila, sobre o Postgres,
// com retry, jitter e chave de idempotência; mais um pool para os
// workflows, o disparo dos agendamentos e o resgate das tentativas
// órfãs — todos sob o mesmo errgroup, que cancela os outros quando um
// falha.
func (w *Worker) Run(ctx context.Context) error {
	s := w.c.store.RelogioDoBanco()
	g, ctx := errgroup.WithContext(ctx)
	filas := map[string]map[string]runner.Handler{}
	for q := range w.cfg.Queues {
		filas[q] = w.handlers
	}
	d := delivery.Novo(s)
	if w.cfg.BreakerLimiar > 0 {
		d.Breakers.Limiar = w.cfg.BreakerLimiar
	}
	if w.cfg.BreakerPausa > 0 {
		d.Breakers.Pausa = w.cfg.BreakerPausa
	}
	filas[webhook.FanoutQueue] = map[string]runner.Handler{
		webhook.FanoutKind: d.Fanout, webhook.DeliverKind: d.Entregar}
	if len(w.workflows) > 0 {
		r := &wf.Replayer{Store: s, Funcs: w.workflows, Now: time.Now}
		filas[workflow.AdvanceQueue] = map[string]runner.Handler{
			workflow.AdvanceKind: r.Handler()}
	}
	retry := policy.Retry{Base: w.cfg.RetryBase, Max: w.cfg.RetryMax}
	for q, hs := range filas {
		p := &worker.Pool{
			Queue: postgres.NewFila(ctx, s), QueueName: q,
			Handlers: hs, Concurrency: max(w.cfg.Queues[q], 2),
			PollTimeout:    5 * time.Second,
			AttemptTimeout: w.cfg.RescueAfter,
			Backoff: func(a int) time.Duration {
				return retry.Delay(a, rand.Float64)
			},
			ReportEvery: time.Minute, Worker: w.cfg.Name,
			Now: time.Now, Log: w.cfg.Log,
		}
		g.Go(func() error { return p.Run(ctx) })
	}
	g.Go(func() error {
		return aCada(ctx, time.Second, func(agora time.Time) error {
			_, _, err := s.FireDue(ctx, agora, schedule.Disparo)
			return err
		})
	})
	g.Go(func() error {
		return aCada(ctx, w.cfg.RescueAfter/4,
			func(agora time.Time) error {
				_, err := s.Rescue(ctx, agora,
					agora.Add(-w.cfg.RescueAfter))
				return err
			})
	})
	if err := g.Wait(); !errors.Is(err, context.Canceled) {
		return err
	}
	return nil
}

// livro:fim worker

// aCada chama f a cada intervalo, até o contexto terminar.
func aCada(
	ctx context.Context,
	intervalo time.Duration,
	f func(time.Time) error,
) error {
	t := time.NewTicker(intervalo)
	defer t.Stop()
	for {
		if err := f(time.Now()); err != nil {
			return err
		}
		select {
		case <-t.C:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}
