//go:build integration

package integration_test

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/go-sob-pressao/enxame/internal/engine/partition"
	"github.com/go-sob-pressao/enxame/internal/store"
	"github.com/go-sob-pressao/enxame/internal/store/postgres"
	"github.com/go-sob-pressao/enxame/test/testutil"
)

// A posse vence, outro nó adquire com um token maior, e o Store cercado
// do dono antigo passa a recusar tudo — reservar, promover, resgatar.
func TestCercaRecusaODonoAntigo(t *testing.T) {
	db := testutil.Postgres(t)
	s := postgres.New(db)
	enfileirar(t, s, 3)
	ctx := t.Context()
	a := partition.Lease{DB: db, No: "a", Duracao: time.Second}
	b := partition.Lease{DB: db, No: "b", Duracao: time.Second}
	ta, ok, err := a.Adquirir(ctx, 0)
	if !ok || err != nil {
		t.Fatalf("a: %v %v", ok, err)
	}
	if _, ok, _ := b.Adquirir(ctx, 0); ok {
		t.Fatal("b adquiriu uma posse vigente")
	}
	if _, err := db.Exec(ctx, `UPDATE partition_lease
		SET lease_expires_at = now() - interval '1 s'
		WHERE partition_id = 0`); err != nil { // a posse de a venceu
		t.Fatal(err)
	}
	tb, ok, err := b.Adquirir(ctx, 0)
	if !ok || err != nil || tb <= ta {
		t.Fatalf("b: token %d depois de %d: %v %v", tb, ta, ok, err)
	}
	velho := s.ComCerca(postgres.Cerca{Particao: 0, RangeID: ta})
	_, _, err = velho.Claim(ctx, "q", time.Now(), "a")
	if !errors.Is(err, store.ErrCercado) {
		t.Fatalf("a reservou com o token antigo: %v", err)
	}
	if _, err := velho.Promote(ctx, time.Now()); !errors.Is(err,
		store.ErrCercado) {
		t.Fatalf("a promoveu com o token antigo: %v", err)
	}
	if ok, _ := a.Renovar(ctx, 0, ta); ok {
		t.Fatal("a renovou uma posse que não é mais dela")
	}
	novo := s.ComCerca(postgres.Cerca{Particao: 0, RangeID: tb})
	if _, achou, err := novo.Claim(ctx, "q", time.Now(), "b"); !achou ||
		err != nil {
		t.Fatalf("b não reservou: %v %v", achou, err)
	}
}

// Dois Donos disputam a partição: um só trabalha por vez, e quando ele
// para, o outro assume, com um token maior.
func TestDonoTrocaDeMao(t *testing.T) {
	db := testutil.Postgres(t)
	var tokens sync.Map
	var ativos atomic.Int32
	trabalho := func(no string) func(context.Context, int64) error {
		return func(ctx context.Context, token int64) error {
			if n := ativos.Add(1); n > 1 {
				t.Errorf("%d donos ao mesmo tempo", n)
			}
			defer ativos.Add(-1)
			tokens.Store(no, token)
			<-ctx.Done()
			return nil
		}
	}
	dono := func(no string) partition.Dono {
		return partition.Dono{Particao: 0, Log: slog.New(slog.DiscardHandler),
			Lease: partition.Lease{DB: db, No: no,
				Duracao: 900 * time.Millisecond}}
	}
	ctxA, pararA := context.WithCancel(t.Context())
	ctxB, pararB := context.WithCancel(t.Context())
	defer pararB()
	var wg sync.WaitGroup
	wg.Go(func() { _ = dono("a").Run(ctxA, trabalho("a")) })
	esperar(t, func() bool { _, ok := tokens.Load("a"); return ok })
	wg.Go(func() { _ = dono("b").Run(ctxB, trabalho("b")) })
	pararA() // a solta a partição ao parar
	esperar(t, func() bool { _, ok := tokens.Load("b"); return ok })
	ta, _ := tokens.Load("a")
	tb, _ := tokens.Load("b")
	if tb.(int64) <= ta.(int64) {
		t.Fatalf("token de b %d não é maior que o de a %d", tb, ta)
	}
	pararB()
	wg.Wait()
}

func esperar(t *testing.T, cond func() bool) {
	t.Helper()
	limite := time.After(10 * time.Second)
	for !cond() {
		select {
		case <-limite:
			t.Fatal("condição não chegou em 10 s")
		case <-time.After(20 * time.Millisecond):
		}
	}
}
