package queue_test

import (
	"context"
	"testing"
	"time"

	"github.com/go-sob-pressao/enxame/internal/core/job"
	"github.com/go-sob-pressao/enxame/internal/queue"
)

func TestFeedEntregaTodosEFechaAoCancelar(t *testing.T) {
	m := queue.NewMemory()
	for i := range 5 {
		if _, err := m.Insert(job.Spec{ID: jid(byte(i + 1)), Queue: "q", Kind: "eco"}, t0); err != nil {
			t.Fatal(err)
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	jobs := queue.Feed(ctx, m, queue.FeedConfig{
		Queue: "q", Worker: "w", Idle: time.Millisecond, Now: func() time.Time { return t0 },
	})
	for range 5 {
		j := <-jobs
		if j.State != job.StateRunning {
			t.Fatalf("job entregue em %q", j.State)
		}
	}
	cancel()
	for range jobs { // o canal precisa fechar: só assim o range termina
	}
}
