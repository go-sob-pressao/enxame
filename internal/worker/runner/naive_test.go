package runner_test

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"
	"uuid"

	"github.com/go-sob-pressao/enxame/internal/core/id"
	"github.com/go-sob-pressao/enxame/internal/core/job"
	"github.com/go-sob-pressao/enxame/internal/queue"
	"github.com/go-sob-pressao/enxame/internal/worker/runner"
)

// livro:inicio naive-carga

// O worker ingênuo não tem limite: n jobs em espera viram n goroutines
// vivas. O handler segura cada job até todos terem começado, e o teste
// mede o pico.
func TestNaiveUmaGoroutinePorJob(t *testing.T) {
	const n = 1000
	q := queue.NewMemory()
	agora := time.Date(2026, 9, 24, 12, 0, 0, 0, time.UTC)
	for range n {
		s := job.Spec{
			ID:    id.JobID(uuid.NewV7()),
			Queue: "q",
			Kind:  "segura",
		}
		if _, err := q.Insert(s, agora); err != nil {
			t.Fatal(err)
		}
	}

	var emExecucao, pico atomic.Int64
	var todosComecaram sync.WaitGroup
	todosComecaram.Add(n)
	w := &runner.Naive{
		Queue: q, QueueName: "q", Worker: "w1",
		Now: func() time.Time { return agora },
		Handlers: map[string]runner.Handler{
			"segura": func(context.Context, job.Job) error {
				registrarPico(&pico, emExecucao.Add(1))
				todosComecaram.Done()
				todosComecaram.Wait()
				emExecucao.Add(-1)
				return nil
			},
		},
	}
	executados, err := w.Drain(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if executados != n || pico.Load() != n {
		t.Fatalf(
			"executados=%d pico=%d; esperado %d e %d",
			executados,
			pico.Load(),
			n,
			n,
		)
	}
	t.Logf(
		"%d jobs, %d goroutines de handler vivas ao mesmo tempo",
		n,
		pico.Load(),
	)
}

// livro:fim naive-carga

// registrarPico guarda em pico o maior valor já visto.
func registrarPico(pico *atomic.Int64, atual int64) {
	for {
		p := pico.Load()
		if atual <= p || pico.CompareAndSwap(p, atual) {
			return
		}
	}
}
