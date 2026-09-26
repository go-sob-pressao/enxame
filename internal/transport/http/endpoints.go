package http

import (
	"net/http"

	"github.com/go-sob-pressao/enxame/internal/core/webhook"
	"github.com/go-sob-pressao/enxame/internal/store"
)

var errNaoEncontrado = store.ErrNotFound

// livro:inicio endpoint

// criarEndpoint inscreve um endpoint de webhook. O corpo é validado
// pelo domínio antes de chegar ao banco, e o segredo nunca passa por
// aqui: o cliente manda uma referência a ele — env:NOME, por exemplo —,
// e só quem assina a entrega a resolve.
func (a *API) criarEndpoint(w http.ResponseWriter, r *http.Request) {
	var n NovoEndpoint
	if err := ler(r, &n); err != nil {
		a.erro(w, r, err)
		return
	}
	e := webhook.Endpoint{Namespace: namespace(r.Context()),
		URL: n.URL, Description: n.Description,
		EventTypes: n.EventTypes, SecretRef: n.SecretRef}
	if err := webhook.Validar(e); err != nil {
		a.erro(w, r, err)
		return
	}
	e, err := a.Store.CreateEndpoint(r.Context(), e)
	if err != nil {
		a.erro(w, r, err)
		return
	}
	escrever(w, http.StatusCreated, paraEndpoint(e))
}

// livro:fim endpoint

func (a *API) listarEndpoints(w http.ResponseWriter, r *http.Request) {
	es, err := a.Store.ListEndpoints(
		r.Context(),
		namespace(r.Context()),
	)
	if err != nil {
		a.erro(w, r, err)
		return
	}
	lista := Lista[Endpoint]{Items: make([]Endpoint, 0, len(es))}
	for _, e := range es {
		lista.Items = append(lista.Items, paraEndpoint(e))
	}
	escrever(w, http.StatusOK, lista)
}

func (a *API) desativarEndpoint(
	w http.ResponseWriter,
	r *http.Request,
) {
	err := a.Store.DisableEndpoint(r.Context(), namespace(r.Context()),
		r.PathValue("id"), "desativado pela API")
	if err != nil {
		a.erro(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func paraEndpoint(e webhook.Endpoint) Endpoint {
	return Endpoint{ID: e.ID, URL: e.URL, Description: e.Description,
		EventTypes: e.EventTypes, SecretRef: e.SecretRef,
		Disabled: e.Disabled, CreatedAt: e.CreatedAt.UTC()}
}
