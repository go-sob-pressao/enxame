package liderperdido_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/go-sob-pressao/enxame/internal/cluster/coordinator"
	"github.com/go-sob-pressao/enxame/internal/core/id"
	"github.com/go-sob-pressao/enxame/internal/core/job"
	"github.com/go-sob-pressao/enxame/internal/store/postgres"
	liderperdido "github.com/go-sob-pressao/enxame/missoes/05-lider-perdido"
	"github.com/go-sob-pressao/enxame/test/testutil"
)

// livro:inicio missao-05-teste

// Três nós dividem as 512 partições. Chegam 120 jobs agendados para
// daqui a um segundo, e o líder cai logo em seguida. Os outros dois
// assumem as partições dele. Em 15 segundos, os 120 jobs precisam
// estar disponíveis.
func TestMissao(t *testing.T) {
	db := testutil.Postgres(t)
	nos := map[coordinator.NodeID]*liderperdido.No{}
	for i := 1; i <= 3; i++ {
		n := liderperdido.Subir(db, coordinator.NodeID(fmt.Sprint("n", i)))
		nos[n.Nome] = n
		defer n.Parar()
	}
	esperar(t, 20*time.Second, func() bool {
		total := 0
		for _, n := range nos {
			total += len(n.Posses.Tokens())
		}
		return total == coordinator.NumPartitions
	})
	s := postgres.New(db)
	hora := time.Now().Add(time.Second)
	for i := range 120 {
		jid, _ := id.ParseJobID(fmt.Sprintf(
			"0192a3b4-0000-7000-8000-%012d", i+1))
		evs, j, err := job.Insert(job.Spec{ID: jid, Namespace: "ns",
			Queue: "q", Kind: "eco", RunAt: hora}, time.Now())
		if err != nil {
			t.Fatal(err)
		}
		j, _ = job.ApplyAll(j, evs)
		if err := s.Insert(t.Context(), j, evs); err != nil {
			t.Fatal(err)
		}
	}
	lider, _ := nos["n1"].Coord.Leader(context.Background())
	nos[lider].Parar()
	t.Logf("líder %s derrubado com 120 jobs agendados", lider)
	agendados := func() int {
		var n int
		_ = db.QueryRow(t.Context(), `SELECT count(*) FROM job
			WHERE state = 'scheduled'`).Scan(&n)
		return n
	}
	limite := time.Now().Add(15 * time.Second)
	for agendados() > 0 && time.Now().Before(limite) {
		<-time.After(200 * time.Millisecond)
	}
	for nome, n := range nos {
		if nome != lider {
			t.Logf("%s tem %d partições", nome, len(n.Posses.Tokens()))
		}
	}
	var semDono int
	_ = db.QueryRow(t.Context(), `SELECT count(*) FROM partition_lease
		WHERE owner = $1 OR owner IS NULL`, string(lider)).Scan(&semDono)
	t.Logf("partições ainda do líder morto ou sem dono: %d", semDono)
	if n := agendados(); n > 0 {
		t.Fatalf("%d jobs continuam agendados, 15 s depois da hora",
			n)
	}
}

// livro:fim missao-05-teste

func esperar(t *testing.T, prazo time.Duration, cond func() bool) {
	t.Helper()
	limite := time.Now().Add(prazo)
	for !cond() {
		if time.Now().After(limite) {
			t.Fatal("o cluster não dividiu as partições a tempo")
		}
		<-time.After(100 * time.Millisecond)
	}
}
