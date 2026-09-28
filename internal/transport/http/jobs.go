package http

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5"

	"github.com/go-sob-pressao/enxame/internal/core/id"
	"github.com/go-sob-pressao/enxame/internal/core/job"
	"github.com/go-sob-pressao/enxame/internal/observ/tracing"
	"github.com/go-sob-pressao/enxame/internal/store"
)

// brutos são argumentos que chegam prontos em JSON.
type brutos struct {
	kind string
	json []byte
}

func (b brutos) Kind() string                 { return b.kind }
func (b brutos) MarshalJSON() ([]byte, error) { return b.json, nil }

var _ json.Marshaler = brutos{}

// livro:inicio inserir-job

// inserirJob enfileira o job do corpo e responde com ele. Os argumentos
// já chegam em JSON válido — o decodificador conferiu —, e seguem como
// estão até o banco; a resposta é o job que acabou de ser gravado, sem
// lê-lo de volta. Cada cópia a menos de args é uma a menos por
// requisição (Cap. 29).
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
	cheia, err := a.filaCheia(r.Context(), namespace(r.Context()))
	if err != nil {
		a.erro(w, r, err)
		return
	}
	if cheia {
		recusarFilaCheia(w)
		return
	}
	j, evs, err := novoJob(namespace(r.Context()), n,
		tracing.TraceParent(r.Context()))
	if err != nil {
		a.erro(w, r, fmt.Errorf("%w: %w", errEntrada, err))
		return
	}
	ctx := r.Context()
	err = pgx.BeginFunc(ctx, a.DB, func(tx pgx.Tx) error {
		return a.Store.InsertTx(ctx, tx, j, evs)
	})
	if err != nil {
		a.erro(w, r, err)
		return
	}
	escrever(w, http.StatusCreated, paraJob(j))
}

// novoJob monta o job e o evento de criação. Os instantes vão com a
// precisão do banco, o microssegundo, para a resposta ser o que uma
// leitura devolveria.
func novoJob(ns string, n NovoJob,
	traco string) (job.Job, []job.Event, error) {
	args := []byte(n.Args)
	if len(args) == 0 {
		args = []byte("{}")
	}
	spec := job.Spec{ID: id.JobID(uuid.NewV7()), Namespace: ns,
		Queue: n.Queue, Kind: n.Kind, Args: args,
		UniqueKey: n.UniqueKey, OrderingKey: n.OrderingKey,
		RunAt:       n.RunAt.Round(time.Microsecond),
		TraceParent: traco}
	evs, j, err := job.Insert(spec, time.Now().Round(time.Microsecond))
	if err != nil {
		return job.Job{}, nil, err
	}
	j, err = job.ApplyAll(j, evs)
	return j, evs, err
}

// livro:fim inserir-job

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
