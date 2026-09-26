package workflow_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/go-sob-pressao/enxame/internal/store/memory"
	wf "github.com/go-sob-pressao/enxame/internal/worker/workflow"
	"github.com/go-sob-pressao/enxame/internal/worker/workflow/workflowtest"
	"github.com/go-sob-pressao/enxame/pkg/job"
	"github.com/go-sob-pressao/enxame/pkg/workflow"
)

func TestReplayEmMemoria(t *testing.T) {
	workflowtest.Run(t, func(*testing.T) workflowtest.Store {
		return memory.NewWorkflows()
	})
}

// Dentro de um passo, a chave de idempotência é a da posição, e não
// muda entre a execução que falhou e a que concluiu.
func TestChaveDoPassoEstavel(t *testing.T) {
	s := memory.NewWorkflows()
	var chaves []string
	fn := func(c *workflow.Context, _ json.RawMessage) (any, error) {
		return workflow.Step(c, "cobrar",
			func(ctx context.Context) (int, error) {
				chaves = append(chaves, job.IdempotencyKey(ctx))
				if len(chaves) == 1 {
					return 0, errors.New("timeout")
				}
				return 1, nil
			})
	}
	r := &wf.Replayer{Store: s, Funcs: map[string]wf.Func{"f": fn},
		Now: time.Now}
	runID, err := s.StartRun(t.Context(), workflow.Run{Type: "f"})
	if err != nil {
		t.Fatal(err)
	}
	_, _ = r.Advance(t.Context(), runID)
	if _, err := r.Advance(t.Context(), runID); err != nil {
		t.Fatal(err)
	}
	esperada := "step:" + runID + ":1"
	if len(chaves) != 2 || chaves[0] != esperada || chaves[1] != esperada {
		t.Fatalf("chaves %v; esperada %s", chaves, esperada)
	}
}
