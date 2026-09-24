package runner_test

import (
	"context"
	"errors"
	"testing"
	"time"
	"uuid"

	"github.com/go-sob-pressao/enxame/internal/core/id"
	"github.com/go-sob-pressao/enxame/internal/core/job"
	"github.com/go-sob-pressao/enxame/internal/queue"
	"github.com/go-sob-pressao/enxame/internal/worker/runner"
)

func TestDrainExecutaTodosEmOrdem(t *testing.T) {
	agora := time.Date(2026, 9, 24, 12, 0, 0, 0, time.UTC)
	q := queue.NewMemory()
	var ids []id.JobID
	for range 3 {
		s := job.Spec{
			ID:    id.JobID(uuid.NewV7()),
			Queue: "q",
			Kind:  "eco",
		}
		if _, err := q.Insert(s, agora); err != nil {
			t.Fatal(err)
		}
		ids = append(ids, s.ID)
		agora = agora.Add(time.Millisecond)
	}
	falhou := false
	var vistos []id.JobID
	r := &runner.Sequential{
		Queue: q, QueueName: "q", Worker: "w1", RetryDelay: 0,
		Now: func() time.Time { return agora },
		Handlers: map[string]runner.Handler{
			"eco": func(_ context.Context, j job.Job) error {
				vistos = append(vistos, j.ID)
				if !falhou && j.ID == ids[1] {
					falhou = true
					return errors.New("instável")
				}
				return nil
			},
		},
	}
	n, err := r.Drain(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if n != 4 || len(vistos) != 4 {
		t.Fatalf(
			"executou %d vezes (%v); esperado 4: três jobs e um retry",
			n,
			vistos,
		)
	}
	for _, jid := range ids {
		j, _, _ := q.Get(jid)
		if j.State != job.StateCompleted {
			t.Errorf("job %s em %q", jid, j.State)
		}
	}
}
