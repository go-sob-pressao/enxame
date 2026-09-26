package worker_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/go-sob-pressao/enxame/internal/core/job"
	"github.com/go-sob-pressao/enxame/internal/queue"
	"github.com/go-sob-pressao/enxame/internal/worker/runner"
	pkgjob "github.com/go-sob-pressao/enxame/pkg/job"
)

// Três tentativas do mesmo job veem a mesma chave de idempotência, e
// números de tentativa diferentes.
func TestChaveIgualEmTodasAsTentativas(t *testing.T) {
	q := queue.NewMemory()
	var mu sync.Mutex
	var vistas []pkgjob.Info
	h := func(ctx context.Context, _ job.Job) error {
		i, _ := pkgjob.FromContext(ctx)
		mu.Lock()
		defer mu.Unlock()
		vistas = append(vistas, i)
		if len(vistas) < 3 {
			return errors.New("gateway fora do ar")
		}
		return nil
	}
	ids := inserir(t, q, 1, "cobrar")
	p := novoPool(q, map[string]runner.Handler{"cobrar": h}, 1)
	ctx, cancel := context.WithCancel(t.Context())
	fim := make(chan error, 1)
	go func() { fim <- p.Run(ctx) }()
	esperarFinal(t, q, ids)
	cancel()
	<-fim
	if len(vistas) != 3 {
		t.Fatalf("%d tentativas", len(vistas))
	}
	for i, v := range vistas {
		if v.IdempotencyKey != vistas[0].IdempotencyKey ||
			v.Attempt != i+1 || v.IdempotencyKey == "" {
			t.Fatalf("tentativa %d: %+v", i+1, v)
		}
	}
}

// O handler pede para repetir só depois de 300 ms; o pool respeita,
// mesmo com o RetryDelay de 1 ms.
func TestRepetirEm(t *testing.T) {
	q := queue.NewMemory()
	var mu sync.Mutex
	var quando []time.Time
	h := func(context.Context, job.Job) error {
		mu.Lock()
		defer mu.Unlock()
		quando = append(quando, time.Now())
		if len(quando) == 1 {
			return &runner.RepetirEm{Depois: 300 * time.Millisecond,
				Err: errors.New("429 Too Many Requests")}
		}
		return nil
	}
	ids := inserir(t, q, 1, "cobrar")
	p := novoPool(q, map[string]runner.Handler{"cobrar": h}, 1)
	ctx, cancel := context.WithCancel(t.Context())
	fim := make(chan error, 1)
	go func() { fim <- p.Run(ctx) }()
	esperarFinal(t, q, ids)
	cancel()
	<-fim
	if len(quando) != 2 || quando[1].Sub(quando[0]) < 300*time.Millisecond {
		t.Fatalf("tentativas em %v", quando)
	}
}
