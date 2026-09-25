package lacuna_test

import (
	"context"
	"slices"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/go-sob-pressao/enxame/examples/cap14/lacuna"
	"github.com/go-sob-pressao/enxame/test/testutil"
)

func banco(t *testing.T) *pgxpool.Pool {
	t.Helper()
	db := testutil.Postgres(t)
	if _, err := db.Exec(t.Context(), lacuna.Esquema); err != nil {
		t.Fatal(err)
	}
	return db
}

func iniciar(t *testing.T, db *pgxpool.Pool) pgx.Tx {
	t.Helper()
	tx, err := db.Begin(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = tx.Rollback(context.Background()) })
	return tx
}

// esperarBloqueio espera alguma sessão parar à espera de um lock.
func esperarBloqueio(t *testing.T, db *pgxpool.Pool) {
	t.Helper()
	for range 500 {
		var n int
		err := db.QueryRow(t.Context(), `SELECT count(*)
			FROM pg_stat_activity WHERE wait_event_type = 'Lock'`,
		).Scan(&n)
		if err != nil {
			t.Fatal(err)
		}
		if n > 0 {
			return
		}
		<-time.After(10 * time.Millisecond)
	}
	t.Fatal("nenhuma sessão bloqueada em 5 s")
}

// A mesma intercalação contra o fluxo com next_seq: a segunda transação
// espera a primeira, e o leitor recebe os dois eventos, em ordem.
func TestFluxoNaoPulaEvento(t *testing.T) {
	db := banco(t)
	ctx := t.Context()
	_, err := db.Exec(ctx, `INSERT INTO fluxo (id) VALUES (1)`)
	if err != nil {
		t.Fatal(err)
	}
	leitor := &lacuna.LeitorDoFluxo{Fluxo: 1}

	t1 := iniciar(t, db)
	if err := lacuna.GravarNoFluxo(ctx, t1, 1, "a"); err != nil {
		t.Fatal(err)
	}
	feito := make(chan error, 1)
	go func() {
		t2, err := db.Begin(ctx)
		if err == nil {
			err = lacuna.GravarNoFluxo(ctx, t2, 1, "b")
		}
		if err == nil {
			err = t2.Commit(ctx)
		}
		feito <- err
	}()
	esperarBloqueio(t, db) // t2 parada no UPDATE de fluxo
	if lidos, _ := leitor.Ler(ctx, db); len(lidos) != 0 {
		t.Fatalf("leu %v antes de qualquer COMMIT", lidos)
	}
	if err := t1.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	if err := <-feito; err != nil {
		t.Fatal(err)
	}
	lidos, err := leitor.Ler(ctx, db)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(lidos, []string{"a", "b"}) {
		t.Fatalf("leu %v; esperado [a b]", lidos)
	}
}

// A sequence não volta no ROLLBACK: o id alocado pela transação que
// desistiu fica vago para sempre.
func TestSequenceDeixaLacunaNoRollback(t *testing.T) {
	db := banco(t)
	ctx := t.Context()
	tx := iniciar(t, db)
	if err := lacuna.Gravar(ctx, tx, "desistiu"); err != nil {
		t.Fatal(err)
	}
	if err := tx.Rollback(ctx); err != nil {
		t.Fatal(err)
	}
	var id int64
	err := db.QueryRow(ctx, `INSERT INTO evento_global (dado)
		VALUES ('ficou') RETURNING id`).Scan(&id)
	if err != nil {
		t.Fatal(err)
	}
	if id != 2 {
		t.Fatalf("id %d; esperado 2, com o 1 vago", id)
	}
}
