package versao_test

import (
	"testing"
	"time"

	"github.com/go-sob-pressao/enxame/examples/cap17/versao"
	"github.com/go-sob-pressao/enxame/internal/store/memory"
	wf "github.com/go-sob-pressao/enxame/internal/worker/workflow"
	"github.com/go-sob-pressao/enxame/pkg/workflow"
)

type cenario struct {
	s     *memory.Workflows
	agora time.Time
	f     versao.Feitos
}

func novo() *cenario {
	return &cenario{s: memory.NewWorkflows(),
		agora: time.Date(2026, 9, 24, 14, 0, 0, 0, time.UTC)}
}

func (c *cenario) avancar(
	t *testing.T,
	fn wf.Func,
	runID string,
) (wf.Outcome, error) {
	r := &wf.Replayer{Store: c.s, Funcs: map[string]wf.Func{"p": fn},
		Now: func() time.Time { return c.agora }}
	return r.Advance(t.Context(), runID)
}

func (c *cenario) iniciar(t *testing.T) string {
	t.Helper()
	id, err := c.s.StartRun(t.Context(), workflow.Run{Type: "p"})
	if err != nil {
		t.Fatal(err)
	}
	return id
}

// O run de ontem, com o código de hoje e workflow.Version: segue o
// caminho antigo e termina.
func TestRunAntigoComVersion(t *testing.T) {
	c := novo()
	id := c.iniciar(t)
	if _, err := c.avancar(t, c.f.Antes, id); err != nil {
		t.Fatal(err)
	}
	c.agora = c.agora.Add(49 * time.Hour) // deploy, e a entrega chega
	o, err := c.avancar(t, c.f.ComVersion, id)
	if err != nil || o.State != workflow.RunCompleted {
		t.Fatalf("%+v, %v", o, err)
	}
	t.Logf("passos executados: %v", c.f)
}
