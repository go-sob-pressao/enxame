package raftcoord_test

import (
	"fmt"
	"testing"
	"time"

	"go.uber.org/goleak"

	"github.com/go-sob-pressao/enxame/internal/cluster/coordinator"
	"github.com/go-sob-pressao/enxame/internal/cluster/raftcoord"
)

func TestMain(m *testing.M) {
	goleak.VerifyTestMain(m,
		// timers de entrega da rede em memória ainda em voo ao fim
		goleak.IgnoreTopFunction("time.goFunc"))
}

type grupo struct {
	rede   *raftcoord.Rede
	coords map[coordinator.NodeID]*raftcoord.Coord
	ids    []coordinator.NodeID
}

func (g *grupo) Membros() []coordinator.NodeID { return g.ids }

func (g *grupo) Coordenador(
	id coordinator.NodeID,
) coordinator.Coordinator {
	return g.coords[id]
}
func (g *grupo) Parar(id coordinator.NodeID) {
	_ = g.coords[id].Close()
	delete(g.coords, id)
}

func novo(t *testing.T, n int) coordinator.Grupo {
	g := &grupo{
		rede:   raftcoord.NovaRede(time.Millisecond),
		coords: map[coordinator.NodeID]*raftcoord.Coord{},
	}
	for i := 1; i <= n; i++ {
		g.ids = append(g.ids, coordinator.NodeID(fmt.Sprintf("n%d", i)))
	}
	for _, id := range g.ids {
		g.coords[id] = raftcoord.New(raftcoord.Config{
			ID: id, Peers: g.ids, Tick: 5 * time.Millisecond,
			ElectionTicks: 10, HeartbeatTicks: 2, Rede: g.rede,
		})
	}
	t.Cleanup(func() {
		for _, c := range g.coords {
			_ = c.Close()
		}
	})
	return g
}

// livro:inicio raftcoord-conformidade

func TestConformidade(t *testing.T) {
	coordinator.Conformidade(t, novo, 5*time.Second)
}

// livro:fim raftcoord-conformidade
