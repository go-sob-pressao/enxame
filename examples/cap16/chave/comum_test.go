package chave_test

import (
	"context"
	"encoding/json"
	"log/slog"
	"sync/atomic"
	"testing"
	"time"
	"uuid"

	"github.com/go-sob-pressao/enxame/examples/cap16/chave"
	"github.com/go-sob-pressao/enxame/internal/core/id"
	"github.com/go-sob-pressao/enxame/internal/core/job"
	"github.com/go-sob-pressao/enxame/internal/queue"
	"github.com/go-sob-pressao/enxame/internal/worker"
	"github.com/go-sob-pressao/enxame/internal/worker/runner"
)

// cincoTentativas roda uma cobrança com até cinco tentativas, num pool
// sobre a fila em memória, e devolve quantas tentativas houve.
func cincoTentativas(
	t *testing.T,
	h func(context.Context, job.Job) error,
) int64 {
	t.Helper()
	q := queue.NewMemory()
	args, _ := json.Marshal(chave.Assinatura{Cliente: "c-42", Valor: 90})
	jid := id.JobID(uuid.NewV7())
	if _, err := q.Insert(job.Spec{ID: jid, Queue: "q", Kind: "cobrar",
		Args: args, MaxAttempts: 5}, time.Now()); err != nil {
		t.Fatal(err)
	}
	var tentativas atomic.Int64
	contar := func(ctx context.Context, j job.Job) error {
		tentativas.Add(1)
		return h(ctx, j)
	}
	ctx, cancel := context.WithCancel(t.Context())
	p := &worker.Pool{Queue: q, QueueName: "q", Concurrency: 1,
		Handlers:    map[string]runner.Handler{"cobrar": contar},
		PollTimeout: 50 * time.Millisecond, AttemptTimeout: time.Second,
		RetryDelay: time.Millisecond, ReportEvery: time.Hour,
		Worker: "w", Now: time.Now, Log: slog.New(slog.DiscardHandler)}
	fim := make(chan error, 1)
	go func() { fim <- p.Run(ctx) }()
	for {
		j, _, _ := q.Get(jid)
		if j.State == job.StateCompleted || j.State == job.StateDiscarded {
			break
		}
		<-time.After(5 * time.Millisecond)
	}
	cancel()
	<-fim
	return tentativas.Load()
}
