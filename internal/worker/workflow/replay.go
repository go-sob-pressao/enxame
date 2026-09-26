package workflow

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/go-sob-pressao/enxame/pkg/workflow"
)

// Store é o que o replay precisa do armazenamento de runs.
type Store interface {
	// LoadRun devolve o run e seus passos gravados, em ordem.
	LoadRun(ctx context.Context, runID string) (
		workflow.Run, []workflow.Record, error)
	// AppendStep grava o próximo passo e, se next não for zero, o job
	// que continua o run, na mesma transação; store.ErrConflict se a
	// posição já foi gravada, ou se o run já foi encerrado.
	AppendStep(
		ctx context.Context,
		runID string,
		r workflow.Record,
		next workflow.Continuation,
	) error
	// CloseRun encerra o run com a saída ou o erro.
	CloseRun(ctx context.Context, run workflow.Run) error
}

// Func é uma função de workflow registrada.
type Func func(c *workflow.Context, input json.RawMessage) (any, error)

// Outcome é o resultado de um avanço: o estado do run e, se ele ficou
// suspenso, até quando.
type Outcome struct {
	State workflow.RunState
	Until time.Time
	// Gravou diz se esta execução gravou algum passo novo.
	Gravou bool
}

// Replayer avança runs reexecutando a função desde o início.
type Replayer struct {
	Store Store
	Funcs map[string]Func
	Now   func() time.Time
}

// livro:inicio replay

// Advance reexecuta o run desde o primeiro passo. Os passos gravados
// devolvem o resultado da memória; o primeiro passo não gravado executa
// de verdade. A execução termina de um de quatro jeitos: a função
// devolve (o run é encerrado), um Sleep suspende (o run espera), o
// código diverge do histórico (o run para, sem ser encerrado), ou um
// erro transitório interrompe (a próxima execução tenta de novo).
func (r *Replayer) Advance(
	ctx context.Context,
	runID string,
) (Outcome, error) {
	return r.avancar(ctx, runID, 0)
}

// AdvanceAt é o Advance de um job: só a posição dada pode executar pela
// primeira vez, e cada passo novo é gravado junto com o job que executa
// a posição seguinte.
func (r *Replayer) AdvanceAt(
	ctx context.Context,
	runID string,
	posicao int,
) (Outcome, error) {
	return r.avancar(ctx, runID, posicao)
}

func (r *Replayer) avancar(
	ctx context.Context,
	runID string,
	posicao int,
) (Outcome, error) {
	run, passos, err := r.Store.LoadRun(ctx, runID)
	if err != nil {
		return Outcome{}, err
	}
	if run.State != workflow.RunRunning {
		return Outcome{State: run.State}, nil
	}
	fn, ok := r.Funcs[run.Type]
	if !ok {
		return Outcome{}, fmt.Errorf("workflow %q não registrado",
			run.Type)
	}
	h := &historico{store: r.Store, runID: runID, passos: passos,
		continuar: posicao > 0, agora: r.Now}
	c := workflow.NewContext(ctx, runID, h, r.Now, posicao)
	saida, err := fn(c, run.Input)

	var suspenso *workflow.SuspendedError
	var passo *workflow.StepError
	switch {
	case err == nil:
		if run.Output, err = json.Marshal(saida); err != nil {
			return Outcome{}, err
		}
		run.State = workflow.RunCompleted
	case errors.As(err, &suspenso):
		return Outcome{State: run.State, Until: suspenso.Until,
			Gravou: len(h.passos) > len(passos)}, nil
	case errors.As(err, &passo):
		run.State, run.Err = workflow.RunFailed, err.Error()
	default: // transitório ou não determinístico: o run continua aberto
		return Outcome{State: run.State}, err
	}
	return Outcome{State: run.State}, r.Store.CloseRun(ctx, run)
}

// livro:fim replay

// historico é a visão do workflow sobre os passos de um run: lê o que
// foi carregado e grava no Store o que é novo.
type historico struct {
	store     Store
	runID     string
	passos    []workflow.Record
	continuar bool // cada passo novo leva o job da posição seguinte
	agora     func() time.Time
}

func (h *historico) Lookup(seq int) (workflow.Record, bool) {
	if seq < 1 || seq > len(h.passos) {
		return workflow.Record{}, false
	}
	return h.passos[seq-1], true
}

func (h *historico) Append(
	ctx context.Context,
	r workflow.Record,
) error {
	var next workflow.Continuation
	if h.continuar {
		next = workflow.Continuation{Seq: r.Seq + 1, At: h.agora()}
		if r.Kind == workflow.KindSleep {
			next.At = r.WakeAt
		}
	}
	if err := h.store.AppendStep(ctx, h.runID, r, next); err != nil {
		return err
	}
	h.passos = append(h.passos, r)
	return nil
}
