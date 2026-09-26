// Command coord é a bancada da ADR-003: pgcoord × raftcoord, com tempos
// de detecção equivalentes (~1 s nos dois).
//
//	make up
//	ENXAME_DB_DSN=postgres://postgres:enxame@localhost:5432/enxame \
//	    go run ./test/load/coord -rodadas 20
//
// Mede: (1) failover — derrubar o líder e esperar os outros concordarem
// num líder novo e num mapa sem ele; (2) carga no banco em regime, em
// transações por segundo (pg_stat_database); (3) um nó cai enquanto o
// Postgres está pausado (docker pause). O pgcoord usa o membership do
// Capítulo 23, com batidas a cada 100 ms e folga de 350 ms.
package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"slices"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/go-sob-pressao/enxame/internal/cluster/coordinator"
	"github.com/go-sob-pressao/enxame/internal/cluster/membership"
	"github.com/go-sob-pressao/enxame/internal/cluster/pgcoord"
	"github.com/go-sob-pressao/enxame/internal/cluster/raftcoord"
	"github.com/go-sob-pressao/enxame/internal/store/postgres"
)

var (
	rodadas   = flag.Int("rodadas", 20, "repetições do failover")
	container = flag.String(
		"container",
		"enxame-postgres-1",
		"contêiner do Postgres",
	)
	ids = []coordinator.NodeID{"n1", "n2", "n3"}
)

type grupo struct {
	coords map[coordinator.NodeID]coordinator.Coordinator
	parar  map[coordinator.NodeID]func() // derruba um nó inteiro
}

func (g *grupo) fechar() {
	for id := range g.coords {
		g.derrubar(id)
	}
}

func (g *grupo) derrubar(id coordinator.NodeID) {
	if p, ok := g.parar[id]; ok {
		p()
	} else {
		_ = g.coords[id].Close()
	}
	delete(g.coords, id)
}

func main() {
	flag.Parse()
	ctx := context.Background()
	pool, err := preparar(ctx, os.Getenv("ENXAME_DB_DSN"))
	if err != nil {
		fmt.Fprintln(os.Stderr, "banco:", err)
		os.Exit(1)
	}
	defer pool.Close()

	fmt.Println("| Medida | pgcoord | raftcoord |")
	fmt.Println("|---|---|---|")
	fp := failover(func() *grupo { return comPg(ctx, pool) })
	fr := failover(comRaft)
	fmt.Printf(
		"| Failover, mediana (%d rodadas) | %s | %s |\n",
		*rodadas,
		p(fp, 50),
		p(fr, 50),
	)
	fmt.Printf("| Failover, p90 | %s | %s |\n", p(fp, 90), p(fr, 90))
	fmt.Printf("| Failover, pior | %s | %s |\n", p(fp, 100), p(fr, 100))
	fmt.Printf(
		"| Transações no banco por segundo, 3 nós | %.0f | 0 |\n",
		carga(ctx, pool),
	)
	fmt.Printf(
		"| Nó cai com o Postgres pausado 5 s: até o mapa excluí-lo"+
			" | %s | %s |\n",
		comBancoFora(func() *grupo { return comPg(ctx, pool) }),
		comBancoFora(comRaft),
	)
}

// preparar recria o banco da bancada e o migra.
func preparar(ctx context.Context, dsn string) (*pgxpool.Pool, error) {
	const nome = "enxame_bancada_coord"
	adm, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, err
	}
	defer adm.Close()
	for _, sql := range []string{
		"DROP DATABASE IF EXISTS " + nome + " WITH (FORCE)",
		"CREATE DATABASE " + nome,
	} {
		if _, err := adm.Exec(ctx, sql); err != nil {
			return nil, err
		}
	}
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, err
	}
	cfg.ConnConfig.Database = nome
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, err
	}
	return pool, postgres.Migrate(ctx, pool)
}

func limpar(ctx context.Context, pool *pgxpool.Pool) {
	_, _ = pool.Exec(ctx, `TRUNCATE cluster_member, coord_assignment;
		UPDATE coord_leader
		   SET term = 0, node = NULL, expires_at = '-infinity'`)
}

func comPg(ctx context.Context, pool *pgxpool.Pool) *grupo {
	limpar(ctx, pool)
	g := &grupo{
		coords: map[coordinator.NodeID]coordinator.Coordinator{},
		parar:  map[coordinator.NodeID]func(){},
	}
	for _, id := range ids {
		ctxNo, cancel := context.WithCancel(ctx)
		m := membership.Novo(membership.Config{No: id, DB: pool,
			Intervalo: 100 * time.Millisecond,
			Detector: func() membership.Detector {
				p := membership.NovoPhi(100 * time.Millisecond)
				p.Folga = 350 * time.Millisecond
				return p
			}})
		fim := make(chan struct{})
		go func() { _ = m.Run(ctxNo); close(fim) }()
		c := pgcoord.New(ctxNo, pgcoord.Config{
			ID: id, Pool: pool, Membros: m,
			Intervalo: 100 * time.Millisecond, Lease: time.Second,
			Log: slog.New(slog.DiscardHandler),
		})
		g.coords[id] = c
		g.parar[id] = func() { _ = c.Close(); cancel(); <-fim }
	}
	return g
}

func comRaft() *grupo {
	rede := raftcoord.NovaRede(time.Millisecond)
	g := &grupo{
		coords: map[coordinator.NodeID]coordinator.Coordinator{},
	}
	for _, id := range ids {
		g.coords[id] = raftcoord.New(raftcoord.Config{
			ID: id, Peers: ids, Tick: 50 * time.Millisecond,
			ElectionTicks: 10, HeartbeatTicks: 2, Rede: rede,
		})
	}
	return g
}

// convergiu: todos os vivos veem o mesmo líder (vivo) e um mapa cujos
// donos são exatamente os vivos.
func convergiu(
	ctx context.Context,
	g *grupo,
	vivos []coordinator.NodeID,
) bool {
	var lider coordinator.NodeID
	var epoca uint64
	for i, id := range vivos {
		l, _ := g.coords[id].Leader(ctx)
		a, _ := g.coords[id].Assignment(ctx)
		if l == "" || !slices.Contains(vivos, l) ||
			(i > 0 && (l != lider || a.Epoch != epoca)) {
			return false
		}
		donos := slices.Compact(slices.Sorted(slices.Values(a.Owners)))
		if !slices.Equal(donos, vivos) {
			return false
		}
		lider, epoca = l, a.Epoch
	}
	return true
}

func esperar(
	ctx context.Context,
	g *grupo,
	vivos []coordinator.NodeID,
	limite time.Duration,
) time.Duration {
	inicio := time.Now()
	for time.Since(inicio) < limite {
		if convergiu(ctx, g, vivos) {
			return time.Since(inicio)
		}
		<-time.After(5 * time.Millisecond)
	}
	return -1
}

func failover(novo func() *grupo) []time.Duration {
	var tempos []time.Duration
	for range *rodadas {
		g := novo()
		esperar(context.Background(), g, ids, 10*time.Second)
		lider, _ := g.coords[ids[0]].Leader(context.Background())
		g.derrubar(lider)
		vivos := slices.DeleteFunc(
			slices.Clone(ids),
			func(id coordinator.NodeID) bool { return id == lider },
		)
		d := esperar(context.Background(), g, vivos, 30*time.Second)
		tempos = append(tempos, d)
		g.fechar()
	}
	slices.Sort(tempos)
	return tempos
}

// carga mede as transações por segundo no banco da bancada, em regime,
// com os três nós do pgcoord no ar: batidas, leituras, lease e mapa.
func carga(ctx context.Context, pool *pgxpool.Pool) float64 {
	g := comPg(ctx, pool)
	defer g.fechar()
	esperar(ctx, g, ids, 10*time.Second)
	transacoes := func() int64 {
		var n int64
		_ = pool.QueryRow(ctx, `SELECT pg_stat_force_next_flush()`).
			Scan(new(any))
		_ = pool.QueryRow(ctx, `SELECT xact_commit + xact_rollback
			FROM pg_stat_database WHERE datname = current_database()`).
			Scan(&n)
		return n
	}
	antes, t0 := transacoes(), time.Now()
	<-time.After(10 * time.Second)
	return float64(transacoes()-antes) / time.Since(t0).Seconds()
}

func comBancoFora(novo func() *grupo) string {
	g := novo()
	defer g.fechar()
	esperar(context.Background(), g, ids, 10*time.Second)
	vitima := ids[2]
	l, _ := g.coords[ids[0]].Leader(context.Background())
	if l == vitima {
		vitima = ids[1]
	}
	docker("pause")
	inicio := time.Now()
	g.derrubar(vitima)
	vivos := slices.DeleteFunc(
		slices.Clone(ids),
		func(id coordinator.NodeID) bool { return id == vitima },
	)
	go func() {
		<-time.After(5 * time.Second)
		docker("unpause")
	}()
	d := esperar(context.Background(), g, vivos, 30*time.Second)
	if d < 0 {
		return "não convergiu"
	}
	<-time.After(time.Until(inicio.Add(6 * time.Second)))
	return d.Round(10 * time.Millisecond).String()
}

func docker(acao string) {
	ctx := context.Background()
	_ = exec.CommandContext(ctx, "docker", acao, *container).Run()
}

func p(ts []time.Duration, pct int) string {
	if len(ts) == 0 {
		return "—"
	}
	i := min(len(ts)-1, (len(ts)*pct+99)/100-1)
	return ts[max(i, 0)].Round(10 * time.Millisecond).String()
}
