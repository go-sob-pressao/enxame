// Command rotulos mostra os rótulos de goroutine no traceback (Go
// 1.27): três jobs presos num handler que não volta, e um SIGQUIT.
//
//	go run ./examples/cap30/rotulos
//
// Depois de um segundo, o processo manda SIGQUIT para si mesmo, e o
// runtime imprime a pilha de cada goroutine — com os rótulos do job no
// cabeçalho das goroutines dos handlers.
package main

import (
	"context"
	"log/slog"
	"os"
	"syscall"
	"time"
	"uuid"

	"github.com/go-sob-pressao/enxame/internal/core/id"
	"github.com/go-sob-pressao/enxame/internal/core/job"
	"github.com/go-sob-pressao/enxame/internal/queue"
	"github.com/go-sob-pressao/enxame/internal/worker"
	"github.com/go-sob-pressao/enxame/internal/worker/runner"
)

func main() {
	q := queue.NewMemory()
	for _, kind := range []string{"fatura.gerar", "email.enviar",
		"fatura.gerar"} {
		if _, err := q.Insert(job.Spec{ID: id.JobID(uuid.NewV7()),
			Queue: "q", Kind: kind}, time.Now()); err != nil {
			panic(err)
		}
	}
	preso := func(ctx context.Context, _ job.Job) error {
		<-ctx.Done() // um handler que espera o que nunca chega
		return ctx.Err()
	}
	p := &worker.Pool{Queue: q, QueueName: "q", Concurrency: 3,
		Handlers: map[string]runner.Handler{
			"fatura.gerar": preso, "email.enviar": preso},
		PollTimeout: time.Second, AttemptTimeout: time.Hour,
		RetryDelay: time.Second, ReportEvery: time.Hour, Worker: "w1",
		Now: time.Now, Log: slog.New(slog.DiscardHandler)}
	go func() {
		<-time.After(time.Second)
		_ = syscall.Kill(os.Getpid(), syscall.SIGQUIT)
	}()
	_ = p.Run(context.Background())
}
