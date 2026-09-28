package pgcoord_test

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/goleak"

	"github.com/go-sob-pressao/enxame/internal/cluster/coordinator"
	"github.com/go-sob-pressao/enxame/internal/cluster/membership"
	"github.com/go-sob-pressao/enxame/internal/cluster/pgcoord"
	"github.com/go-sob-pressao/enxame/test/testutil"
)

func TestMain(m *testing.M) {
	goleak.VerifyTestMain(m, goleak.IgnoreTopFunction(
		"github.com/jackc/pgx/v5/pgxpool.(*Pool).backgroundHealthCheck",
	))
}

// no é um nó do teste: o membership e o coordenador, que param juntos.
type no struct {
	coord *pgcoord.Coord
	parar func()
}

type grupo struct {
	mu  sync.Mutex
	nos map[coordinator.NodeID]no
	ids []coordinator.NodeID
}

func (g *grupo) Membros() []coordinator.NodeID { return g.ids }

func (g *grupo) Coordenador(
	id coordinator.NodeID,
) coordinator.Coordinator {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.nos[id].coord
}

func (g *grupo) Parar(id coordinator.NodeID) {
	g.mu.Lock()
	n := g.nos[id]
	g.mu.Unlock()
	n.parar()
}

// detector rápido, para o teste não esperar a folga de produção.
func detector() membership.Detector {
	p := membership.NovoPhi(50 * time.Millisecond)
	p.Folga = 200 * time.Millisecond
	return p
}

func novo(t *testing.T, n int) coordinator.Grupo {
	db := testutil.Postgres(t)
	g := &grupo{nos: map[coordinator.NodeID]no{}}
	for i := 1; i <= n; i++ {
		id := coordinator.NodeID(fmt.Sprintf("n%d", i))
		g.ids = append(g.ids, id)
		m := membership.Novo(membership.Config{No: id, DB: db,
			Intervalo: 50 * time.Millisecond, Detector: detector})
		ctx, cancel := context.WithCancel(context.Background())
		var wg sync.WaitGroup
		wg.Go(func() { _ = m.Run(ctx) })
		c := pgcoord.New(ctx, pgcoord.Config{ID: id, Pool: db,
			Membros: m, Intervalo: 50 * time.Millisecond,
			Lease: 300 * time.Millisecond,
			Log:   slog.New(slog.DiscardHandler)})
		var uma sync.Once
		g.nos[id] = no{coord: c, parar: func() {
			uma.Do(func() { _ = c.Close(); cancel(); wg.Wait() })
		}}
	}
	t.Cleanup(func() {
		for _, id := range g.ids {
			g.Parar(id)
		}
	})
	return g
}

// livro:inicio pgcoord-conformidade

func TestConformidade(t *testing.T) {
	coordinator.Conformidade(t, novo, 10*time.Second)
}

// livro:fim pgcoord-conformidade

// livro:inicio lider-reinicia

// O líder reinicia com o mesmo nome — um pod de StatefulSet — antes de
// o lease vencer: o processo novo já é o líder. Nenhum mapa pode dar
// as partições todas a ele porque ele ainda não viu os outros.
func TestLiderReiniciaComOMesmoNome(t *testing.T) {
	db := testutil.Postgres(t)
	subir := func(id coordinator.NodeID) func() {
		m := membership.Novo(membership.Config{No: id, DB: db,
			Intervalo: 50 * time.Millisecond, Detector: detector})
		ctx, cancel := context.WithCancel(context.Background())
		var wg sync.WaitGroup
		wg.Go(func() { _ = m.Run(ctx) })
		c := pgcoord.New(ctx, pgcoord.Config{ID: id, Pool: db,
			Membros: m, Intervalo: 50 * time.Millisecond,
			Lease: time.Second, Log: slog.New(slog.DiscardHandler)})
		return func() { _ = c.Close(); cancel(); wg.Wait() }
	}
	paradas := map[coordinator.NodeID]func(){}
	for _, id := range []coordinator.NodeID{"n1", "n2", "n3"} {
		paradas[id] = subir(id)
	}
	defer func() {
		for _, p := range paradas {
			p()
		}
	}()
	var lider coordinator.NodeID
	var epoca int64
	prazo := time.Now().Add(10 * time.Second)
	for time.Now().Before(prazo) {
		_ = db.QueryRow(t.Context(), `SELECT l.node, max(a.epoch)
			FROM coord_leader l, coord_assignment a GROUP BY l.node`).
			Scan(&lider, &epoca)
		if donosDistintos(t, db, epoca) == 3 {
			break
		}
		<-time.After(50 * time.Millisecond)
	}
	paradas[lider]()
	paradas[lider] = subir(lider) // antes de o lease de 1 s vencer
	// Uma janela de observação, não uma sincronização: o que se confere
	// é que nada de errado aconteceu nela.
	<-time.After(2 * time.Second)
	var ultima int64
	_ = db.QueryRow(t.Context(),
		`SELECT max(epoch) FROM coord_assignment`).Scan(&ultima)
	for e := epoca + 1; e <= ultima; e++ {
		if n := donosDistintos(t, db, e); n < 3 {
			t.Errorf("época %d: %d dono(s) para 512 partições", e, n)
		}
	}
}

// livro:fim lider-reinicia

func donosDistintos(t *testing.T, db *pgxpool.Pool, epoca int64) int {
	t.Helper()
	var n int
	_ = db.QueryRow(t.Context(), `SELECT count(DISTINCT d)
		FROM coord_assignment a, jsonb_array_elements_text(a.owners) d
		WHERE a.epoch = $1`, epoca).Scan(&n)
	return n
}
