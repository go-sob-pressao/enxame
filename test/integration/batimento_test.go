//go:build integration

package integration_test

import (
	"context"
	"log/slog"
	"sync/atomic"
	"testing"
	"time"

	"github.com/go-sob-pressao/enxame/internal/core/job"
	"github.com/go-sob-pressao/enxame/internal/store/postgres"
	"github.com/go-sob-pressao/enxame/internal/worker"
	"github.com/go-sob-pressao/enxame/internal/worker/runner"
	"github.com/go-sob-pressao/enxame/test/testutil"
)

// cenarioLongo roda um job cuja primeira tentativa dura dois segundos,
// com um resgatador de prazo de 700 ms a cada 100 ms. Devolve o job
// final e o histórico.
func cenarioLongo(
	t *testing.T,
	comBatimento bool,
) (job.Job, []job.Event) {
	t.Helper()
	s := postgres.New(testutil.Postgres(t))
	ids := enfileirar(t, s, 1)
	var tentativas atomic.Int64
	h := func(ctx context.Context, _ job.Job) error {
		if tentativas.Add(1) > 1 {
			return nil // a segunda tentativa é rápida
		}
		select {
		case <-time.After(2 * time.Second):
			return nil
		case <-ctx.Done():
			return context.Cause(ctx)
		}
	}
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	f := postgres.NewFila(ctx, s)
	p := &worker.Pool{Queue: f, QueueName: "q", Concurrency: 2,
		Handlers:    map[string]runner.Handler{"eco": h},
		PollTimeout: time.Second, AttemptTimeout: 10 * time.Second,
		RetryDelay: time.Millisecond, ReportEvery: time.Hour,
		Worker: "w1", Now: time.Now, Log: slog.New(slog.DiscardHandler)}
	if comBatimento {
		p.Heartbeat, p.HeartbeatEvery = f.Heartbeat, 200*time.Millisecond
	}
	fim := make(chan error, 1)
	go func() { fim <- p.Run(ctx) }()
	go func() {
		for ctx.Err() == nil {
			agora := time.Now()
			_, _ = s.Rescue(ctx, agora, agora.Add(-700*time.Millisecond))
			<-time.After(100 * time.Millisecond)
		}
	}()
	esperarConcluidos(t, s, ids)
	<-time.After(2500 * time.Millisecond) // o fim da primeira tentativa
	select {
	case err := <-fim:
		t.Fatalf("o pool parou: %v", err)
	default:
	}
	j, _, _ := s.Get(t.Context(), ids[0])
	hist, _ := s.History(t.Context(), ids[0])
	return j, hist
}

func contarEventos(evs []job.Event, tipo job.EventType) int {
	n := 0
	for _, e := range evs {
		if e.Type == tipo {
			n++
		}
	}
	return n
}

// livro:inicio batimento-teste

// Com batimento a cada 200 ms, a tentativa de dois segundos nunca fica
// 700 ms sem sinal de vida, e não é resgatada.
func TestBatimentoEvitaResgate(t *testing.T) {
	j, hist := cenarioLongo(t, true)
	if j.Attempt != 1 || contarEventos(hist, job.EventRescued) != 0 {
		t.Fatalf("tentativa %d, %d resgates", j.Attempt,
			contarEventos(hist, job.EventRescued))
	}
	t.Logf("batimentos: %d", contarEventos(hist, job.EventHeartbeat))
}

// Sem batimento, a tentativa longa é resgatada viva; a segunda termina
// o job, e o fim tardio da primeira é recusado sem derrubar o pool.
func TestSemBatimentoResgataViva(t *testing.T) {
	j, hist := cenarioLongo(t, false)
	if j.Attempt != 2 || contarEventos(hist, job.EventRescued) != 1 {
		t.Fatalf("tentativa %d, %d resgates", j.Attempt,
			contarEventos(hist, job.EventRescued))
	}
}

// livro:fim batimento-teste
