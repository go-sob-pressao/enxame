//go:build integration

package integration_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/go-sob-pressao/enxame/internal/core/id"
	"github.com/go-sob-pressao/enxame/internal/core/job"
	"github.com/go-sob-pressao/enxame/internal/engine/partition"
	"github.com/go-sob-pressao/enxame/internal/store/postgres"
	"github.com/go-sob-pressao/enxame/test/testutil"
)

// livro:inicio custo-cerca

// O custo da cerca: uma reserva e uma conclusão por iteração, com e sem
// o SELECT … FOR SHARE no começo de cada transação.
func BenchmarkCerca(b *testing.B) {
	for _, comCerca := range []bool{false, true} {
		b.Run(fmt.Sprintf("cerca=%v", comCerca), func(b *testing.B) {
			db := testutil.Postgres(b)
			s := postgres.New(db)
			if comCerca {
				l := partition.Lease{DB: db, No: "a",
					Duracao: time.Hour}
				token, _, err := l.Adquirir(b.Context(), 0)
				if err != nil {
					b.Fatal(err)
				}
				s = s.ComCerca(postgres.Cerca{Particao: 0,
					RangeID: token})
			}
			agora := time.Now()
			for i := range b.N {
				jid, _ := id.ParseJobID(fmt.Sprintf(
					"0192a3b4-0000-7000-8000-%012d", i+1))
				evs, j, _ := job.Insert(job.Spec{ID: jid,
					Namespace: "ns", Queue: "q", Kind: "eco"}, agora)
				j, _ = job.ApplyAll(j, evs)
				if err := s.Insert(b.Context(), j, evs); err != nil {
					b.Fatal(err)
				}
			}
			f := postgres.NewFila(b.Context(), s)
			b.ResetTimer()
			for range b.N {
				j, ok, err := f.Fetch("q", agora, "w")
				if err != nil || !ok {
					b.Fatal(ok, err)
				}
				if err := f.Complete(j.ID, j.Attempt,
					agora); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// livro:fim custo-cerca

// A transação mínima do motor: uma promoção sem nada para promover. O
// que sobra é o custo da própria transação — e o da cerca.
func BenchmarkCercaTransacaoVazia(b *testing.B) {
	for _, comCerca := range []bool{false, true} {
		b.Run(fmt.Sprintf("cerca=%v", comCerca), func(b *testing.B) {
			db := testutil.Postgres(b)
			s := postgres.New(db)
			if comCerca {
				l := partition.Lease{DB: db, No: "a",
					Duracao: time.Hour}
				token, _, err := l.Adquirir(b.Context(), 0)
				if err != nil {
					b.Fatal(err)
				}
				s = s.ComCerca(postgres.Cerca{Particao: 0,
					RangeID: token})
			}
			agora := time.Now()
			for b.Loop() {
				if _, err := s.Promote(b.Context(), agora); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
