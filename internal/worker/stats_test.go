package worker_test

import (
	"context"
	"testing"
	"time"

	"github.com/go-sob-pressao/enxame/internal/core/job"
	"github.com/go-sob-pressao/enxame/internal/queue"
	"github.com/go-sob-pressao/enxame/internal/worker/runner"
)

// O painel lê Stats e a probe lê Running enquanto o pool trabalha.
func TestPainelLeEnquantoPoolTrabalha(t *testing.T) {
	q := queue.NewMemory()
	ids := inserir(t, q, 100, "eco")
	p := novoPool(q, map[string]runner.Handler{
		"eco": func(context.Context, job.Job) error { return nil },
	}, 4)
	parar := rodar(t, p)
	leituras := 0
	for !p.Running() || leituras < 50 {
		_ = p.Stats()
		leituras++
		<-time.After(time.Millisecond)
	}
	esperarFinal(t, q, ids)
	if err := parar(); err != nil {
		t.Fatal(err)
	}
	if s := p.Stats(); s.Executed != 100 {
		t.Fatalf("Executed=%d, esperado 100", s.Executed)
	}
}
