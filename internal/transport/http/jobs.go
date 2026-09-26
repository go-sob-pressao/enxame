package http

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/go-sob-pressao/enxame/internal/core/id"
	"github.com/go-sob-pressao/enxame/internal/core/job"
	"github.com/go-sob-pressao/enxame/internal/store"
	"github.com/go-sob-pressao/enxame/pkg/enxame"
)

// brutos são argumentos que chegam prontos em JSON.
type brutos struct {
	kind string
	json []byte
}

func (b brutos) Kind() string                 { return b.kind }
func (b brutos) MarshalJSON() ([]byte, error) { return b.json, nil }

var _ json.Marshaler = brutos{}

func (a *API) inserirJob(w http.ResponseWriter, r *http.Request) {
	var n NovoJob
	if err := ler(r, &n); err != nil {
		a.erro(w, r, err)
		return
	}
	if n.Queue == "" || n.Kind == "" {
		a.erro(w, r, fmt.Errorf(
			"%w: queue e kind são obrigatórios", errEntrada))
		return
	}
	args := []byte(n.Args)
	if len(args) == 0 {
		args = []byte("{}")
	}
	opts := []enxame.Option{enxame.Queue(n.Queue)}
	if n.UniqueKey != "" {
		opts = append(opts, enxame.UniqueKey(n.UniqueKey))
	}
	if !n.RunAt.IsZero() {
		opts = append(opts, enxame.RunAt(n.RunAt))
	}
	c := enxame.New(a.DB, namespace(r.Context()))
	jid, err := c.Insert(r.Context(), brutos{n.Kind, args}, opts...)
	if err != nil {
		a.erro(w, r, err)
		return
	}
	j, err := a.buscarJob(r.Context(), jid)
	if err != nil {
		a.erro(w, r, err)
		return
	}
	escrever(w, http.StatusCreated, j)
}

// buscarJob lê o job, e o esconde de quem é de outro namespace.
func (a *API) buscarJob(ctx context.Context, s string) (Job, error) {
	jid, err := id.ParseJobID(s)
	if err != nil {
		return Job{}, fmt.Errorf("%w: %w", errEntrada, err)
	}
	j, _, err := a.Store.Get(ctx, jid)
	if err != nil {
		return Job{}, err
	}
	if j.Namespace != namespace(ctx) {
		return Job{}, store.ErrNotFound
	}
	return paraJob(j), nil
}

// livro:inicio espera

// descreverJob devolve o job. Com ?wait=30s, é um long-poll: espera o
// job chegar a um estado final, até o prazo da requisição, o tempo
// pedido — ou o começo do desligamento do servidor, o que vier antes.
func (a *API) descreverJob(w http.ResponseWriter, r *http.Request) {
	espera, _ := time.ParseDuration(r.URL.Query().Get("wait"))
	limite := time.After(espera)
	for {
		j, err := a.buscarJob(r.Context(), r.PathValue("id"))
		if err != nil {
			a.erro(w, r, err)
			return
		}
		if final(j.State) || espera <= 0 {
			escrever(w, http.StatusOK, j)
			return
		}
		select {
		case <-time.After(100 * time.Millisecond):
		case <-limite:
			espera = 0
		case <-a.encerrando: // o servidor vai desligar: responda já
			espera = 0
		case <-r.Context().Done():
			a.erro(w, r, r.Context().Err())
			return
		}
	}
}

// livro:fim espera

func final(estado string) bool {
	switch job.State(estado) {
	case job.StateCompleted, job.StateDiscarded, job.StateCancelled:
		return true
	}
	return false
}

func (a *API) cancelarJob(w http.ResponseWriter, r *http.Request) {
	j, err := a.buscarJob(r.Context(), r.PathValue("id"))
	if err != nil {
		a.erro(w, r, err)
		return
	}
	jid, _ := id.ParseJobID(j.ID)
	novo, err := a.Store.Decide(r.Context(), jid,
		func(j job.Job) ([]job.Event, error) {
			return job.Cancel(j, agora())
		})
	if err != nil {
		a.erro(w, r, err)
		return
	}
	escrever(w, http.StatusOK, paraJob(novo))
}

func paraJob(j job.Job) Job {
	return Job{ID: j.ID.String(), Queue: j.Queue, Kind: j.Kind,
		Args: j.Args, State: string(j.State), Attempt: j.Attempt,
		MaxAttempts: j.MaxAttempts, ScheduledAt: j.ScheduledAt.UTC(),
		LastError: j.LastError, FinalizedAt: j.FinalizedAt.UTC()}
}
