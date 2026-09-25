package memory

import (
	"context"
	"fmt"
	"sync"
	"uuid"

	"github.com/go-sob-pressao/enxame/internal/store"
	"github.com/go-sob-pressao/enxame/pkg/workflow"
)

// Workflows guarda runs e passos em memória.
type Workflows struct {
	mu     sync.Mutex
	runs   map[string]workflow.Run
	passos map[string][]workflow.Record
}

// NewWorkflows cria o armazenamento vazio.
func NewWorkflows() *Workflows {
	return &Workflows{
		runs:   map[string]workflow.Run{},
		passos: map[string][]workflow.Record{},
	}
}

// StartRun cria o run e devolve o id.
func (w *Workflows) StartRun(
	_ context.Context,
	run workflow.Run,
) (string, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	for _, r := range w.runs {
		if r.Namespace == run.Namespace &&
			r.WorkflowID == run.WorkflowID &&
			r.State == workflow.RunRunning {
			return "", fmt.Errorf("%w: run aberto para %s",
				store.ErrDuplicate, run.WorkflowID)
		}
	}
	run.ID, run.State = uuid.NewV7().String(), workflow.RunRunning
	w.runs[run.ID] = run
	return run.ID, nil
}

// LoadRun devolve o run e uma cópia dos passos.
func (w *Workflows) LoadRun(
	_ context.Context,
	runID string,
) (workflow.Run, []workflow.Record, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	run, ok := w.runs[runID]
	if !ok {
		return workflow.Run{}, nil, store.ErrNotFound
	}
	return run, append([]workflow.Record(nil), w.passos[runID]...), nil
}

// AppendStep grava o passo na próxima posição.
func (w *Workflows) AppendStep(
	_ context.Context,
	runID string,
	r workflow.Record,
) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	run, ok := w.runs[runID]
	if !ok {
		return store.ErrNotFound
	}
	if run.State != workflow.RunRunning ||
		r.Seq != len(w.passos[runID])+1 {
		return store.ErrConflict
	}
	w.passos[runID] = append(w.passos[runID], r)
	return nil
}

// CloseRun encerra o run, se ainda estiver aberto.
func (w *Workflows) CloseRun(
	_ context.Context,
	run workflow.Run,
) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	atual, ok := w.runs[run.ID]
	if !ok {
		return store.ErrNotFound
	}
	if atual.State != workflow.RunRunning {
		return store.ErrConflict
	}
	w.runs[run.ID] = run
	return nil
}
