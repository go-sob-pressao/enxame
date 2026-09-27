package http

import (
	"net/http"
	"time"
)

// Rota descreve uma rota da API: o padrão do ServeMux, o handler e o
// que o gerador de OpenAPI precisa saber dela.
type Rota struct {
	Padrao  string // "MÉTODO /caminho/{curinga}"
	Resumo  string
	Entrada any // tipo do corpo, ou nil
	Saida   any // tipo da resposta de sucesso
	Status  int // status de sucesso
	Publica bool
	handler http.HandlerFunc
}

// livro:inicio rotas

// Rotas é a API inteira, numa tabela. O ServeMux do Go 1.22 em diante
// entende método e curinga no padrão: "GET /v1/jobs/{id}" não casa com
// POST, e r.PathValue("id") devolve o curinga. Nenhum roteador de
// terceiros.
func (a *API) Rotas() []Rota {
	return []Rota{
		{Padrao: "POST /v1/jobs", Resumo: "enfileira um job",
			Entrada: NovoJob{}, Saida: Job{}, Status: 201,
			handler: a.inserirJob},
		{Padrao: "GET /v1/jobs/{id}", Resumo: "descreve um job; " +
			"?wait=30s espera o estado final",
			Saida: Job{}, Status: 200, handler: a.descreverJob},
		{Padrao: "POST /v1/jobs/{id}/cancel", Resumo: "cancela um job",
			Saida: Job{}, Status: 200, handler: a.cancelarJob},
		{Padrao: "POST /v1/workflows", Resumo: "inicia um workflow",
			Entrada: NovoWorkflow{}, Saida: Run{}, Status: 201,
			handler: a.iniciarWorkflow},
		{Padrao: "GET /v1/workflows/{id}", Resumo: "estado de um run",
			Saida: Run{}, Status: 200, handler: a.descreverWorkflow},
		{Padrao: "POST /v1/webhooks/endpoints",
			Resumo: "inscreve um endpoint", Entrada: NovoEndpoint{},
			Saida: Endpoint{}, Status: 201, handler: a.criarEndpoint},
		{Padrao: "GET /v1/webhooks/endpoints",
			Resumo: "lista os endpoints", Saida: Lista[Endpoint]{},
			Status: 200, handler: a.listarEndpoints},
		{Padrao: "GET /v1/webhooks/endpoints/{id}/state",
			Resumo: "estado do breaker, no dono da partição",
			Saida:  EstadoEndpoint{}, Status: 200,
			handler: a.estadoEndpoint},
		{Padrao: "DELETE /v1/webhooks/endpoints/{id}",
			Resumo: "desativa um endpoint", Status: 204,
			handler: a.desativarEndpoint},
		{Padrao: "GET /v1/cluster/members",
			Resumo: "quem está no cluster, na visão deste nó",
			Saida:  Lista[MembroCluster]{}, Status: 200,
			handler: a.membrosCluster},
		{Padrao: "GET /healthz", Resumo: "o processo está vivo",
			Status: 200, Publica: true, handler: a.vivo},
		{Padrao: "GET /readyz", Resumo: "aceita tráfego",
			Status: 200, Publica: true, handler: a.pronto},
	}
}

// Handler monta o ServeMux a partir da tabela, com os middlewares.
func (a *API) Handler() http.Handler {
	mux := http.NewServeMux()
	protecao := a.protecao()
	for _, r := range a.Rotas() {
		var h http.Handler = r.handler
		if !r.Publica {
			h = Encadear(h, protecao...)
		}
		h = anotarRota(h)
		mux.Handle(r.Padrao, h)
	}
	return Encadear(mux, a.Recuperar, a.Registrar,
		Prazo(10*time.Second, time.Minute))
}

// livro:fim rotas

// livro:inicio protecao

// protecao é a cadeia das rotas autenticadas, montada uma vez: o
// limite de requisições em curso é do processo, não de cada rota. O
// descarte vem antes de tudo — recusar por excesso não pode custar a
// verificação do token —, e o long-poll fica fora dele: passa quase
// todo o tempo parado, e ocuparia uma vaga por até um minuto.
func (a *API) protecao() []Middleware {
	var ms []Middleware
	if a.MaxEmCurso > 0 {
		ms = append(ms, exceto(emEspera, Descartar(a.MaxEmCurso)))
	}
	ms = append(ms, a.Autenticar)
	if a.Taxa > 0 {
		ms = append(ms, Limitar(a.Taxa, max(a.Rajada, 1)))
	}
	return append(ms, LimitarCorpo(1<<20))
}

// livro:fim protecao

// exceto aplica m só às requisições em que pula(r) é falso.
func exceto(pula func(*http.Request) bool, m Middleware) Middleware {
	return func(prox http.Handler) http.Handler {
		com := m(prox)
		return http.HandlerFunc(func(w http.ResponseWriter,
			r *http.Request) {
			if pula(r) {
				prox.ServeHTTP(w, r)
				return
			}
			com.ServeHTTP(w, r)
		})
	}
}

func emEspera(r *http.Request) bool {
	return r.URL.Query().Has("wait")
}
