package http

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/go-sob-pressao/enxame/internal/cluster/routing"
	"github.com/go-sob-pressao/enxame/internal/core/id"
	"github.com/go-sob-pressao/enxame/internal/core/webhook"
)

// Roteador decide se uma partição é atendida aqui (Cap. 26).
type Roteador interface {
	Decidir(ctx context.Context, particao int,
		epoca uint64) routing.Decisao
}

// Entrega é o que a API consulta do estado em memória da entrega.
type Entrega interface {
	EstadoDoEndpoint(id string) string
}

// EstadoEndpoint é o estado de um endpoint no dono da partição dele.
type EstadoEndpoint struct {
	Endpoint string `json:"endpoint"`
	Particao int    `json:"partition"`
	Epoca    uint64 `json:"epoch"`
	Breaker  string `json:"breaker"`
}

// livro:inicio estado-endpoint

// estadoEndpoint devolve o estado do breaker de um endpoint. O breaker
// vive na memória do nó que entrega para o endpoint — o dono da
// partição dele —, e só ele pode responder. Os outros respondem 421,
// com o endereço do dono e a época do mapa em que ele é dono; o cliente
// repete lá, levando a época no cabeçalho Enxame-Epoca.
func (a *API) estadoEndpoint(w http.ResponseWriter, r *http.Request) {
	e, err := a.Store.Endpoint(r.Context(), r.PathValue("id"))
	if err == nil && e.Namespace != namespace(r.Context()) {
		err = errNaoEncontrado // o endpoint é de outro namespace
	}
	if err != nil {
		a.erro(w, r, err)
		return
	}
	if a.Rota == nil || a.Entrega == nil {
		escrever(w, http.StatusOK, EstadoEndpoint{Endpoint: e.ID,
			Breaker: "desconhecido"})
		return
	}
	p := id.Particao(webhook.ChaveDoEndpoint(e.ID))
	epoca, _ := strconv.ParseUint(r.Header.Get("Enxame-Epoca"), 10, 64)
	d := a.Rota.Decidir(r.Context(), p, epoca)
	switch {
	case d.Esperar:
		recusar(w, http.StatusServiceUnavailable, time.Second,
			Erro{"map_stale", "mapa de partições em troca"})
	case !d.Aqui:
		w.Header().Set("Enxame-Dono", d.Endereco)
		w.Header().Set("Enxame-Epoca", strconv.FormatUint(d.Epoca, 10))
		escrever(w, http.StatusMisdirectedRequest, Erro{
			"not_partition_owner", "a partição " + strconv.Itoa(p) +
				" é de " + d.Dono})
	default:
		escrever(w, http.StatusOK, EstadoEndpoint{Endpoint: e.ID,
			Particao: p, Epoca: d.Epoca,
			Breaker: a.Entrega.EstadoDoEndpoint(e.ID)})
	}
}

// livro:fim estado-endpoint
