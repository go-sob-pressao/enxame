package perdida_test

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/go-sob-pressao/enxame/examples/cap15/perdida"
	"github.com/go-sob-pressao/enxame/test/testutil"
)

func banco(t *testing.T) *pgxpool.Pool {
	t.Helper()
	db := testutil.Postgres(t)
	if _, err := db.Exec(t.Context(), perdida.Esquema); err != nil {
		t.Fatal(err)
	}
	return db
}

func iniciar(t *testing.T, db *pgxpool.Pool, nivel pgx.TxIsoLevel) pgx.Tx {
	t.Helper()
	tx, err := db.BeginTx(t.Context(), pgx.TxOptions{IsoLevel: nivel})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = tx.Rollback(context.Background()) })
	return tx
}

func ler(t *testing.T, tx pgx.Tx) (saldo int, versao int64) {
	t.Helper()
	err := tx.QueryRow(t.Context(),
		`SELECT saldo, version FROM conta WHERE id = 1`).Scan(&saldo, &versao)
	if err != nil {
		t.Fatal(err)
	}
	return saldo, versao
}

func saldoFinal(t *testing.T, db *pgxpool.Pool) int {
	t.Helper()
	var s int
	if err := db.QueryRow(t.Context(),
		`SELECT saldo FROM conta WHERE id = 1`).Scan(&s); err != nil {
		t.Fatal(err)
	}
	return s
}

// livro:inicio perdida

// Dois depósitos de 10, lidos e gravados pela aplicação, em READ
// COMMITTED: cada transação lê 100 e grava 110. Nenhum erro; um
// depósito sumiu.
func TestAtualizacaoPerdida(t *testing.T) {
	db := banco(t)
	ctx := t.Context()
	t1 := iniciar(t, db, pgx.ReadCommitted)
	t2 := iniciar(t, db, pgx.ReadCommitted)
	s1, _ := ler(t, t1)
	s2, _ := ler(t, t2)
	for _, p := range []struct {
		tx    pgx.Tx
		saldo int
	}{{t1, s1}, {t2, s2}} {
		if _, err := p.tx.Exec(ctx, `UPDATE conta SET saldo = $1
			WHERE id = 1`, p.saldo+10); err != nil {
			t.Fatal(err)
		}
		if err := p.tx.Commit(ctx); err != nil {
			t.Fatal(err)
		}
	}
	if s := saldoFinal(t, db); s != 110 {
		t.Fatalf("saldo %d", s)
	}
}

// livro:fim perdida

// O banco soma: o UPDATE de t2, em READ COMMITTED, espera t1 e relê a
// linha já com o depósito dela.
func TestSomaNoBanco(t *testing.T) {
	db := banco(t)
	ctx := t.Context()
	t1 := iniciar(t, db, pgx.ReadCommitted)
	t2 := iniciar(t, db, pgx.ReadCommitted)
	for _, tx := range []pgx.Tx{t1, t2} {
		if _, err := tx.Exec(ctx, `UPDATE conta SET saldo = saldo + 10
			WHERE id = 1`); err != nil {
			t.Fatal(err)
		}
		if err := tx.Commit(ctx); err != nil {
			t.Fatal(err)
		}
	}
	if s := saldoFinal(t, db); s != 120 {
		t.Fatalf("saldo %d", s)
	}
}

// Em REPEATABLE READ, o UPDATE de t2 sobre uma linha que t1 mudou depois
// do início de t2 falha com erro de serialização (40001).
func TestRepeatableReadRecusa(t *testing.T) {
	db := banco(t)
	ctx := t.Context()
	t1 := iniciar(t, db, pgx.RepeatableRead)
	t2 := iniciar(t, db, pgx.RepeatableRead)
	s1, _ := ler(t, t1)
	s2, _ := ler(t, t2)
	if _, err := t1.Exec(ctx, `UPDATE conta SET saldo = $1
		WHERE id = 1`, s1+10); err != nil {
		t.Fatal(err)
	}
	if err := t1.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	_, err := t2.Exec(ctx, `UPDATE conta SET saldo = $1 WHERE id = 1`,
		s2+10)
	var pg *pgconn.PgError
	if !errors.As(err, &pg) || pg.Code != "40001" {
		t.Fatalf("esperava 40001: %v", err)
	}
	t.Log(pg.Message)
}

// livro:inicio perdida-versao

// Lock otimista em READ COMMITTED: a versão lida vai no WHERE. O UPDATE
// de t2 não encontra mais a versão 1, e a aplicação sabe que precisa
// ler de novo.
func TestVersaoRecusa(t *testing.T) {
	db := banco(t)
	ctx := t.Context()
	t1 := iniciar(t, db, pgx.ReadCommitted)
	t2 := iniciar(t, db, pgx.ReadCommitted)
	s1, v1 := ler(t, t1)
	s2, v2 := ler(t, t2)
	depositar := func(tx pgx.Tx, saldo int, v int64) int64 {
		tag, err := tx.Exec(ctx, `UPDATE conta
			SET saldo = $1, version = version + 1
			WHERE id = 1 AND version = $2`, saldo+10, v)
		if err != nil {
			t.Fatal(err)
		}
		if err := tx.Commit(ctx); err != nil {
			t.Fatal(err)
		}
		return tag.RowsAffected()
	}
	if n := depositar(t1, s1, v1); n != 1 {
		t.Fatalf("t1 afetou %d linhas", n)
	}
	if n := depositar(t2, s2, v2); n != 0 {
		t.Fatalf("t2 afetou %d linhas; a versão lida já era velha", n)
	}
}

// livro:fim perdida-versao
