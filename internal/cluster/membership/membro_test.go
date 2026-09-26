package membership_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/go-sob-pressao/enxame/internal/cluster/coordinator"
	"github.com/go-sob-pressao/enxame/internal/cluster/membership"
	"github.com/go-sob-pressao/enxame/test/testutil"
)

// grupo é um cluster de nós no mesmo processo, sobre o mesmo banco.
type grupo struct {
	t     *testing.T
	db    *pgxpool.Pool
	ids   []coordinator.NodeID
	mu    sync.Mutex
	nos   map[coordinator.NodeID]*membership.Membro
	parar map[coordinator.NodeID]func()
}

func (g *grupo) Membros() []coordinator.NodeID { return g.ids }

func (g *grupo) Visao(id coordinator.NodeID) coordinator.Membership {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.nos[id]
}

// Voltar sobe o nó de novo, como um processo novo.
func (g *grupo) Voltar(id coordinator.NodeID) {
	m := membership.Novo(membership.Config{No: id, DB: g.db,
		Intervalo: 200 * time.Millisecond})
	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup
	wg.Go(func() { _ = m.Run(ctx) })
	g.mu.Lock()
	g.nos[id] = m
	g.parar[id] = func() { cancel(); wg.Wait() }
	g.mu.Unlock()
}

func (g *grupo) Parar(id coordinator.NodeID) {
	g.mu.Lock()
	p := g.parar[id]
	g.mu.Unlock()
	p()
}

func novoGrupo(t *testing.T, n int) coordinator.GrupoDeMembros {
	g := &grupo{t: t, db: testutil.Postgres(t),
		nos:   map[coordinator.NodeID]*membership.Membro{},
		parar: map[coordinator.NodeID]func(){}}
	for i := range n {
		id := coordinator.NodeID("no-" + string(rune('a'+i)))
		g.ids = append(g.ids, id)
		g.Voltar(id)
	}
	t.Cleanup(func() {
		for _, id := range g.ids {
			g.Parar(id)
		}
	})
	return g
}

func TestMembershipConformidade(t *testing.T) {
	coordinator.ConformidadeMembros(t, novoGrupo, 10*time.Second)
}
