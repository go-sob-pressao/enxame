package routing

import (
	"context"

	"github.com/go-sob-pressao/enxame/internal/cluster/coordinator"
)

// Rota decide, para uma partição, se a requisição fica neste nó ou vai
// para o dono.
type Rota struct {
	No       coordinator.NodeID
	Coord    coordinator.Coordinator
	Tenho    func(particao int) bool            // posse já adquirida
	Endereco func(no coordinator.NodeID) string // HTTP do nó
}

// Decisao é o que fazer com a requisição.
type Decisao struct {
	Aqui     bool   // este nó é o dono e tem a posse: atenda
	Esperar  bool   // ninguém pode responder agora: tente de novo
	Dono     string // senão, quem é o dono,
	Endereco string // onde ele atende,
	Epoca    uint64 // e em que época do mapa isso vale
}

// livro:inicio rota

// Decidir compara a partição com o mapa local. A requisição traz a
// época do mapa que o cliente já viu — a do redirecionamento anterior,
// se houve —, e um nó com o mapa mais velho que isso não redireciona: o
// dono que ele conhece é o de antes. Ele pede para esperar, e o mapa
// dele alcança o do cliente no próximo ciclo do coordenador.
func (r Rota) Decidir(
	ctx context.Context,
	particao int,
	epocaDoPedido uint64,
) Decisao {
	a, err := r.Coord.Assignment(ctx)
	if err != nil || len(a.Owners) != coordinator.NumPartitions ||
		epocaDoPedido > a.Epoch {
		return Decisao{Esperar: true, Epoca: a.Epoch}
	}
	dono := a.Owners[particao]
	if dono == r.No {
		// O mapa diz que é deste nó; a posse pode ainda não ter vindo.
		return Decisao{Aqui: r.Tenho(particao),
			Esperar: !r.Tenho(particao), Epoca: a.Epoch}
	}
	return Decisao{Dono: string(dono), Endereco: r.Endereco(dono),
		Epoca: a.Epoch}
}

// livro:fim rota
