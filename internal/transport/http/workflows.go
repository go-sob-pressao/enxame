package http

import (
	"fmt"
	"net/http"

	"github.com/go-sob-pressao/enxame/pkg/enxame"
)

func (a *API) iniciarWorkflow(w http.ResponseWriter, r *http.Request) {
	var n NovoWorkflow
	if err := ler(r, &n); err != nil {
		a.erro(w, r, err)
		return
	}
	if n.Type == "" || n.WorkflowID == "" {
		a.erro(w, r, fmt.Errorf(
			"%w: type e workflow_id são obrigatórios", errEntrada))
		return
	}
	var input any = brutos{json: []byte(n.Input)}
	if len(n.Input) == 0 {
		input = struct{}{}
	}
	c := enxame.New(a.DB, namespace(r.Context()))
	runID, err := c.StartWorkflow(r.Context(), n.Type, n.WorkflowID,
		input)
	if err != nil {
		a.erro(w, r, err)
		return
	}
	escrever(w, http.StatusCreated, Run{ID: runID, State: "running"})
}

func (a *API) descreverWorkflow(
	w http.ResponseWriter,
	r *http.Request,
) {
	run, _, err := a.Store.LoadRun(r.Context(), r.PathValue("id"))
	if err == nil && run.Namespace != namespace(r.Context()) {
		err = errNaoEncontrado
	}
	if err != nil {
		a.erro(w, r, err)
		return
	}
	escrever(w, http.StatusOK, Run{ID: run.ID, State: string(run.State),
		Output: []byte(run.Output), Error: run.Err})
}
