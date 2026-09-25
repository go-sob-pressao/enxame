//go:build integration

package integration_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/go-sob-pressao/enxame/internal/core/id"
	"github.com/go-sob-pressao/enxame/internal/core/job"
	"github.com/go-sob-pressao/enxame/internal/store/postgres"
	"github.com/go-sob-pressao/enxame/test/testutil"
)

// historicoLongo grava um job com pelo menos n eventos: um ciclo de
// tentativa, falha e retorno à fila, repetido até completar n.
func historicoLongo(b *testing.B, s *postgres.Store, n int) id.JobID {
	b.Helper()
	jid, _ := id.ParseJobID("0192a3b4-0000-7000-8000-000000000001")
	t := time.Date(2026, 9, 25, 14, 0, 0, 0, time.UTC)
	evs, j, err := job.Insert(job.Spec{ID: jid, Namespace: "ns",
		Queue: "q", Kind: "eco", MaxAttempts: n}, t)
	if err != nil {
		b.Fatal(err)
	}
	j, _ = job.ApplyAll(j, evs)
	if err := s.Insert(b.Context(), j, evs); err != nil {
		b.Fatal(err)
	}
	k := len(evs)
	passos := []func(job.Job) ([]job.Event, error){
		func(j job.Job) ([]job.Event, error) {
			return job.Start(j, t, "w1")
		},
		func(j job.Job) ([]job.Event, error) {
			return job.Fail(j, t, "falhou", false, t)
		},
		func(j job.Job) ([]job.Event, error) {
			return job.MakeAvailable(j, t)
		},
	}
	for i := 0; len(evs) < n; i++ {
		t = t.Add(time.Second)
		novos, err := passos[i%len(passos)](j)
		if err != nil {
			b.Fatal(err)
		}
		if j, err = job.ApplyAll(j, novos); err != nil {
			b.Fatal(err)
		}
		evs = append(evs, novos...)
	}
	// Em transações de até mil eventos: mil UPDATEs da mesma linha numa
	// transação só deixam mil versões mortas dela, que cada UPDATE
	// seguinte precisa atravessar.
	for i := k; i < len(evs); i += 1000 {
		lote := evs[i:min(i+1000, len(evs))]
		_, err := s.Decide(b.Context(), jid,
			func(job.Job) ([]job.Event, error) { return lote, nil })
		if err != nil {
			b.Fatal(err)
		}
	}
	return jid
}

// livro:inicio custo-replay

// BenchmarkReplay compara as duas formas de saber o estado de um job:
// ler a projeção, que cada transição mantém em dia, ou reconstruí-la
// do histórico. A projeção é o snapshot que o Enxame nunca deixa
// envelhecer.
func BenchmarkReplay(b *testing.B) {
	for _, n := range []int{10, 1_000, 100_000} {
		s := postgres.New(testutil.Postgres(b))
		jid := historicoLongo(b, s, n)
		nome := fmt.Sprintf("eventos=%d/", n)
		b.Run(nome+"projecao", func(b *testing.B) {
			for b.Loop() {
				if _, _, err := s.Get(b.Context(), jid); err != nil {
					b.Fatal(err)
				}
			}
		})
		b.Run(nome+"replay", func(b *testing.B) {
			for b.Loop() {
				evs, err := s.History(b.Context(), jid)
				if err != nil {
					b.Fatal(err)
				}
				if _, err := job.ApplyAll(job.Job{}, evs); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// livro:fim custo-replay
