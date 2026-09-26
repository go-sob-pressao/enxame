// Package gateway é um gateway de pagamento falso, para os experimentos
// do Capítulo 16. Como os gateways reais, aceita uma chave de
// idempotência: a mesma chave devolve o mesmo recibo, sem novo débito.
package gateway

import (
	"context"
	"errors"
	"fmt"
	"sync"
)

// ErrTimeout é o que o cliente vê quando a resposta se perde: o débito
// pode ter acontecido ou não.
var ErrTimeout = errors.New("gateway: timeout esperando a resposta")

// Recibo é a confirmação de um débito.
type Recibo struct {
	ID      string
	Cliente string
	Valor   int
}

// Gateway guarda os débitos feitos. PerderRespostas é quantas das
// próximas cobranças debitam e, em seguida, "perdem" a resposta.
type Gateway struct {
	mu              sync.Mutex
	debitos         []Recibo
	porChave        map[string]Recibo
	PerderRespostas int
}

// Cobrar debita valor do cliente. Com chave vazia, cada chamada é um
// débito novo.
func (g *Gateway) Cobrar(
	_ context.Context,
	chave, cliente string,
	valor int,
) (Recibo, error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if r, ok := g.porChave[chave]; ok && chave != "" {
		return r, g.perder()
	}
	r := Recibo{ID: fmt.Sprintf("R-%d", len(g.debitos)+1),
		Cliente: cliente, Valor: valor}
	g.debitos = append(g.debitos, r)
	if chave != "" {
		if g.porChave == nil {
			g.porChave = map[string]Recibo{}
		}
		g.porChave[chave] = r
	}
	return r, g.perder()
}

func (g *Gateway) perder() error {
	if g.PerderRespostas > 0 {
		g.PerderRespostas--
		return ErrTimeout
	}
	return nil
}

// Debitos devolve quantos débitos o gateway fez.
func (g *Gateway) Debitos() int {
	g.mu.Lock()
	defer g.mu.Unlock()
	return len(g.debitos)
}
