package routing_test

import (
	"context"
	"slices"
	"testing"

	"github.com/go-sob-pressao/enxame/internal/cluster/coordinator"
	"github.com/go-sob-pressao/enxame/internal/cluster/routing"
)

// mapa é um coordenador parado numa época, com todas as partições de um
// dono só.
type mapa struct {
	epoca uint64
	dono  coordinator.NodeID
}

func (m *mapa) Assignment(context.Context) (coordinator.Assignment,
	error) {
	owners := make([]coordinator.NodeID, coordinator.NumPartitions)
	for i := range owners {
		owners[i] = m.dono
	}
	return coordinator.Assignment{Epoch: m.epoca, Owners: owners}, nil
}
func (m *mapa) Leader(context.Context) (coordinator.NodeID, error) {
	return "", nil
}
func (m *mapa) Members(context.Context) ([]coordinator.NodeID, error) {
	return nil, nil
}
func (m *mapa) Close() error { return nil }

// dois nós com mapas de épocas diferentes: A já viu a época 7, em que a
// partição é de B; B ainda está na 6, em que ela era de A.
func dois() map[string]routing.Rota {
	end := func(n coordinator.NodeID) string { return string(n) }
	tem := func(int) bool { return false }
	return map[string]routing.Rota{
		"a": {No: "a", Coord: &mapa{7, "b"}, Tenho: tem, Endereco: end},
		"b": {No: "b", Coord: &mapa{6, "a"}, Tenho: tem, Endereco: end},
	}
}

// seguir faz o que um cliente faz: vai de redirecionamento em
// redirecionamento, até 10, levando ou não a época do anterior.
func seguir(nos map[string]routing.Rota, levarEpoca bool) []string {
	var caminho []string
	no, epoca := "a", uint64(0)
	for range 10 {
		caminho = append(caminho, no)
		d := nos[no].Decidir(context.Background(), 42, epoca)
		if d.Aqui || d.Esperar {
			return append(caminho, "esperar")
		}
		no = d.Endereco
		if levarEpoca {
			epoca = d.Epoca
		}
	}
	return caminho
}

// livro:inicio enigma-rota

// Sem a época no pedido, A manda para B, e B manda de volta para A,
// até o cliente desistir.
func TestPingPongSemEpoca(t *testing.T) {
	c := seguir(dois(), false)
	t.Logf("sem a época: %v", c)
	if len(c) != 10 || slices.Contains(c, "esperar") {
		t.Fatalf("esperava o pingue-pongue: %v", c)
	}
}

// Com a época, B vê que o cliente conhece um mapa mais novo que o dele
// e não o manda de volta.
func TestEpocaParaOPingPong(t *testing.T) {
	c := seguir(dois(), true)
	t.Logf("com a época: %v", c)
	if !slices.Equal(c, []string{"a", "b", "esperar"}) {
		t.Fatalf("caminho %v", c)
	}
}

// livro:fim enigma-rota
