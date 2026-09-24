package worker_test

import (
	"context"
	"errors"
	"log/slog"
	"sync/atomic"
	"testing"
	"time"
	"uuid"

	"github.com/go-sob-pressao/enxame/internal/core/id"
	"github.com/go-sob-pressao/enxame/internal/core/job"
	"github.com/go-sob-pressao/enxame/internal/queue"
	"github.com/go-sob-pressao/enxame/internal/worker"
	"github.com/go-sob-pressao/enxame/internal/worker/runner"
)

func novoPool(
	q worker.Queue,
	hs map[string]runner.Handler,
	n int,
) *worker.Pool {
	return &worker.Pool{
		Queue: q, QueueName: "q", Handlers: hs, Concurrency: n,
		PollTimeout: 50 * time.Millisecond, AttemptTimeout: 200 * time.Millisecond,
		RetryDelay: time.Millisecond, ReportEvery: 10 * time.Millisecond,
		Worker: "w1", Now: time.Now, Log: slog.New(slog.DiscardHandler),
	}
}

func inserir(
	t *testing.T,
	q *queue.Memory,
	n int,
	kind string,
) []id.JobID {
	t.Helper()
	var ids []id.JobID
	for range n {
		s := job.Spec{
			ID:          id.JobID(uuid.NewV7()),
			Queue:       "q",
			Kind:        kind,
			MaxAttempts: 3,
		}
		if _, err := q.Insert(s, time.Now()); err != nil {
			t.Fatal(err)
		}
		ids = append(ids, s.ID)
	}
	return ids
}

// esperarFinal espera todos os jobs chegarem a um estado final.
func esperarFinal(t *testing.T, q *queue.Memory, ids []id.JobID) {
	t.Helper()
	limite := time.After(10 * time.Second)
	for {
		finais := 0
		for _, jid := range ids {
			if j, _, _ := q.Get(jid); j.State.Final() {
				finais++
			}
		}
		if finais == len(ids) {
			return
		}
		select {
		case <-limite:
			t.Fatalf(
				"%d de %d jobs em estado final depois de 10s",
				finais,
				len(ids),
			)
		case <-time.After(time.Millisecond):
		}
	}
}

func rodar(t *testing.T, p *worker.Pool) (parar func() error) {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	fim := make(chan error, 1)
	go func() { fim <- p.Run(ctx) }()
	return func() error { cancel(); return <-fim }
}

func TestPoolExecutaTodosComLimite(t *testing.T) {
	q := queue.NewMemory()
	var emExecucao, pico atomic.Int64
	ids := inserir(t, q, 200, "eco")
	p := novoPool(q, map[string]runner.Handler{
		"eco": func(context.Context, job.Job) error {
			atual := emExecucao.Add(1)
			for v := pico.Load(); atual > v && !pico.CompareAndSwap(v, atual); v = pico.Load() {
				continue
			}
			<-time.After(time.Millisecond)
			emExecucao.Add(-1)
			return nil
		},
	}, 8)
	parar := rodar(t, p)
	esperarFinal(t, q, ids)
	if err := parar(); err != nil {
		t.Fatal(err)
	}
	if pico.Load() > 8 {
		t.Fatalf(
			"pico de %d handlers simultâneos; limite 8",
			pico.Load(),
		)
	}
}

func TestPoolRecuperaPanicoETimeout(t *testing.T) {
	q := queue.NewMemory()
	panico := inserir(t, q, 1, "panico")[0]
	lento := inserir(t, q, 1, "lento")[0]
	p := novoPool(q, map[string]runner.Handler{
		"panico": func(context.Context, job.Job) error { panic("índice fora do intervalo") },
		"lento":  func(ctx context.Context, _ job.Job) error { <-ctx.Done(); return ctx.Err() },
	}, 2)
	parar := rodar(t, p)
	esperarFinal(t, q, []id.JobID{panico, lento})
	if err := parar(); err != nil {
		t.Fatal(err)
	}
	if j, _, _ := q.Get(panico); j.State != job.StateDiscarded ||
		j.LastError == "" {
		t.Errorf("job com pânico: %+v", j)
	}
	if j, _, _ := q.Get(lento); j.LastError != worker.ErrAttemptTimeout.Error() {
		t.Errorf("job lento: %q", j.LastError)
	}
}

// filaQuebrada falha em toda busca: o errgroup precisa parar o pool
// todo.
type filaQuebrada struct{ *queue.Memory }

var errBanco = errors.New("conexão com o banco perdida")

func (filaQuebrada) Fetch(
	string,
	time.Time,
	string,
) (job.Job, bool, error) {
	return job.Job{}, false, errBanco
}

func TestPoolParaNoPrimeiroErroDaFila(t *testing.T) {
	p := novoPool(filaQuebrada{queue.NewMemory()}, nil, 4)
	if err := p.Run(context.Background()); !errors.Is(err, errBanco) {
		t.Fatalf("Run devolveu %v, esperado errBanco", err)
	}
}
