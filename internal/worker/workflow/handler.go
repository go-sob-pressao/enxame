package workflow

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/go-sob-pressao/enxame/internal/core/job"
	"github.com/go-sob-pressao/enxame/internal/store"
	"github.com/go-sob-pressao/enxame/internal/worker/runner"
	"github.com/go-sob-pressao/enxame/pkg/workflow"
)

// livro:inicio handler

// Handler é o handler do job workflow.AdvanceKind: cada job executa uma
// posição do histórico de um run. O passo novo é gravado junto com o
// job da posição seguinte (AppendStep); o que sobra ao handler é dizer
// ao pool como a tentativa terminou.
func (r *Replayer) Handler() runner.Handler {
	return func(ctx context.Context, j job.Job) error {
		var a workflow.AdvanceArgs
		if err := json.Unmarshal(j.Args, &a); err != nil {
			return runner.Permanent(err)
		}
		o, err := r.AdvanceAt(ctx, a.RunID, a.Seq)
		var nd *workflow.NonDeterministicError
		switch {
		case err == nil && !o.Gravou && o.Until.After(r.Now()):
			// Chegou antes da hora — relógios diferentes entre quem
			// gravou o Sleep e quem executa: tenta de novo mais tarde.
			return fmt.Errorf("run %s dorme até %s", a.RunID,
				o.Until.Format(time.RFC3339Nano))
		case errors.Is(err, store.ErrConflict):
			// Outra execução gravou a posição: o run seguiu sem esta.
			return nil
		case errors.As(err, &nd):
			// Repetir não resolve; o run fica parado (Capítulo 17).
			return runner.Permanent(err)
		}
		return err // nil, ou transitório: o job tenta de novo
	}
}

// livro:fim handler
