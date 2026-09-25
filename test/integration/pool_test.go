//go:build integration

package integration_test

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/go-sob-pressao/enxame/internal/core/id"
	"github.com/go-sob-pressao/enxame/internal/core/job"
	"github.com/go-sob-pressao/enxame/internal/store/postgres"
	"github.com/go-sob-pressao/enxame/internal/worker"
	"github.com/go-sob-pressao/enxame/internal/worker/runner"
	"github.com/go-sob-pressao/enxame/test/testutil"
)

var _ worker.Queue = (*postgres.Fila)(nil)

func enfileirar(t *testing.T, s *postgres.Store, n int) []id.JobID {
	t.Helper()
	var ids []id.JobID
	for i := range n {
		jid, _ := id.ParseJobID(
			fmt.Sprintf("0192a3b4-0000-7000-8000-%012d", i+1),
		)
		evs, j, err := job.Insert(job.Spec{
			ID: jid, Namespace: "ns", Queue: "q", Kind: "eco",
		}, time.Now())
		if err != nil {
			t.Fatal(err)
		}
		j, _ = job.ApplyAll(j, evs)
		if err := s.Insert(t.Context(), j, evs); err != nil {
			t.Fatal(err)
		}
		ids = append(ids, jid)
	}
	return ids
}

func pool(
	f worker.Queue,
	nome string,
	h runner.Handler,
) *worker.Pool {
	return &worker.Pool{
		Queue: f, QueueName: "q", Concurrency: 4,
		Handlers:       map[string]runner.Handler{"eco": h},
		PollTimeout:    time.Second,
		AttemptTimeout: 10 * time.Second,
		RetryDelay:     time.Millisecond,
		ReportEvery:    time.Hour,
		Worker:         nome, Now: time.Now,
		Log: slog.New(slog.DiscardHandler),
	}
}

// esperarConcluidos espera todos os jobs chegarem a completed.
func esperarConcluidos(
	t *testing.T,
	s *postgres.Store,
	ids []id.JobID,
) {
	t.Helper()
	limite := time.Now().Add(30 * time.Second)
	for time.Now().Before(limite) {
		feitos := 0
		for _, jid := range ids {
			j, _, err := s.Get(t.Context(), jid)
			if err != nil {
				t.Fatal(err)
			}
			if j.State == job.StateCompleted {
				feitos++
			}
		}
		if feitos == len(ids) {
			return
		}
		<-time.After(50 * time.Millisecond)
	}
	contagem := map[job.State]int{}
	for _, jid := range ids {
		j, _, _ := s.Get(t.Context(), jid)
		contagem[j.State]++
	}
	t.Fatalf("jobs não concluídos em 30 s: %v", contagem)
}

// livro:inicio pool-postgres

// Dois pools, em goroutines que fazem o papel de dois processos,
// disputam a mesma fila no Postgres. SKIP LOCKED garante que cada job
// seja reservado por um só: nenhum histórico tem duas tentativas.
func TestDoisPoolsCadaJobUmaVez(t *testing.T) {
	s := postgres.New(testutil.Postgres(t))
	ids := enfileirar(t, s, 200)
	var execucoes atomic.Int64
	h := func(context.Context, job.Job) error {
		execucoes.Add(1)
		return nil
	}
	ctx, cancel := context.WithCancel(t.Context())
	var wg sync.WaitGroup
	for _, nome := range []string{"w1", "w2"} {
		wg.Go(func() {
			_ = pool(postgres.NewFila(ctx, s), nome, h).Run(ctx)
		})
	}
	esperarConcluidos(t, s, ids)
	cancel()
	wg.Wait()
	if n := execucoes.Load(); n != 200 {
		t.Fatalf("%d execuções para 200 jobs", n)
	}
	for _, jid := range ids {
		hist, _ := s.History(t.Context(), jid)
		inicios := 0
		for _, e := range hist {
			if e.Type == job.EventAttemptStart {
				inicios++
			}
		}
		if inicios != 1 {
			t.Fatalf("job %s: %d tentativas", jid, inicios)
		}
	}
}

// livro:fim pool-postgres

// O processo para no meio do lote; outro processo, com outro pool,
// termina o trabalho. Nada se perde: a fila está no banco.
func TestTrabalhoSobreviveAoProcesso(t *testing.T) {
	s := postgres.New(testutil.Postgres(t))
	ids := enfileirar(t, s, 60)
	var feitos atomic.Int64
	lento := func(ctx context.Context, _ job.Job) error {
		select {
		case <-time.After(20 * time.Millisecond):
			feitos.Add(1)
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	ctx1, para := context.WithCancel(t.Context())
	fim := make(chan error, 1)
	go func() {
		fim <- pool(postgres.NewFila(ctx1, s), "w1", lento).Run(ctx1)
	}()
	for feitos.Load() < 10 {
		<-time.After(5 * time.Millisecond)
	}
	para()
	if err := <-fim; err != nil && !errors.Is(err, context.Canceled) {
		t.Logf("primeiro pool terminou com %v", err)
	}
	// As tentativas em curso no cancelamento não conseguiram registrar
	// o fim: a Fila guarda o contexto do pool, que já acabou. Ficaram
	// em running, como ficariam depois de um kill -9. O novo processo
	// roda um resgatador com prazo fixo de um segundo, bem acima dos 20
	// ms de cada tentativa.
	ctx2, cancel := context.WithCancel(t.Context())
	defer cancel()
	go func() {
		_ = pool(postgres.NewFila(ctx2, s), "w2", lento).Run(ctx2)
	}()
	go func() {
		for ctx2.Err() == nil {
			agora := time.Now()
			_, _ = s.Rescue(ctx2, agora, agora.Add(-time.Second))
			<-time.After(100 * time.Millisecond)
		}
	}()
	esperarConcluidos(t, s, ids)
}
