//go:build integration

package integration_test

import (
	"context"
	"encoding/json"
	"log/slog"
	"sync/atomic"
	"testing"
	"time"

	"github.com/go-sob-pressao/enxame/internal/store/postgres"
	"github.com/go-sob-pressao/enxame/internal/worker"
	"github.com/go-sob-pressao/enxame/internal/worker/runner"
	wf "github.com/go-sob-pressao/enxame/internal/worker/workflow"
	"github.com/go-sob-pressao/enxame/pkg/workflow"
	"github.com/go-sob-pressao/enxame/test/testutil"
)

// Um run inteiro avançado por jobs: cada posição do histórico é um job,
// enfileirado na mesma transação que grava a posição anterior.
func TestWorkflowAvancaPorJobs(t *testing.T) {
	db := testutil.Postgres(t)
	s := postgres.New(db)
	var cobrancas, notas atomic.Int64
	pedido := func(c *workflow.Context, _ json.RawMessage) (any, error) {
		recibo, err := workflow.Step(c, "cobrar",
			func(context.Context) (string, error) {
				cobrancas.Add(1)
				return "R-1", nil
			})
		if err != nil {
			return nil, err
		}
		err = workflow.Sleep(c, "esperar", 300*time.Millisecond)
		if err != nil {
			return nil, err
		}
		return workflow.Step(c, "emitir-nota",
			func(context.Context) (string, error) {
				notas.Add(1)
				return "nota de " + recibo, nil
			})
	}
	r := &wf.Replayer{Store: s, Now: time.Now,
		Funcs: map[string]wf.Func{"pedido": pedido}}
	id, err := s.StartRun(t.Context(), workflow.Run{
		Namespace: "ns", WorkflowID: "pedido-7", Type: "pedido",
	})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	p := &worker.Pool{
		Queue: postgres.NewFila(
			ctx,
			s,
		), QueueName: workflow.AdvanceQueue,
		Concurrency: 2,
		Handlers: map[string]runner.Handler{
			workflow.AdvanceKind: r.Handler(),
		},
		PollTimeout: time.Second, AttemptTimeout: 10 * time.Second,
		RetryDelay: time.Millisecond, ReportEvery: time.Hour,
		Worker: "w1", Now: time.Now, Log: slog.New(slog.DiscardHandler),
	}
	go func() { _ = p.Run(ctx) }()
	limite := time.Now().Add(20 * time.Second)
	for {
		run, _, err := s.LoadRun(t.Context(), id)
		if err != nil {
			t.Fatal(err)
		}
		if run.State == workflow.RunCompleted {
			if string(run.Output) != `"nota de R-1"` {
				t.Fatalf("saída %s", run.Output)
			}
			break
		}
		if time.Now().After(limite) {
			rows, _ := db.Query(
				t.Context(),
				`SELECT kind, state, args::text,
				coalesce((SELECT string_agg(e.payload::text, ' | ')
				  FROM job_event e WHERE e.job_id = job.job_id), '')
				FROM job ORDER BY created_at`,
			)
			for rows.Next() {
				var k, st, a, ev string
				_ = rows.Scan(&k, &st, &a, &ev)
				t.Log(k, st, a, ev)
			}
			t.Fatalf("run em %s depois de 20 s", run.State)
		}
		<-time.After(50 * time.Millisecond)
	}
	if cobrancas.Load() != 1 || notas.Load() != 1 {
		t.Fatalf(
			"cobranças %d, notas %d",
			cobrancas.Load(),
			notas.Load(),
		)
	}
	var jobs int
	if err := db.QueryRow(t.Context(), `SELECT count(*) FROM job
		WHERE kind = $1`, workflow.AdvanceKind).Scan(&jobs); err != nil {
		t.Fatal(err)
	}
	t.Logf("jobs de avanço: %d", jobs)
}
