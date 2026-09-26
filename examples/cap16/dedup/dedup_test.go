package dedup_test

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/go-sob-pressao/enxame/examples/cap16/dedup"
	"github.com/go-sob-pressao/enxame/test/testutil"
)

func banco(t *testing.T) *pgxpool.Pool {
	t.Helper()
	db := testutil.Postgres(t)
	if _, err := db.Exec(t.Context(), dedup.Esquema+`
		CREATE TABLE ponto (cliente TEXT, pontos INT);`); err != nil {
		t.Fatal(err)
	}
	return db
}

func TestEfeitoUmaVezPorChave(t *testing.T) {
	d := &dedup.Dedup{DB: banco(t)}
	enviados := 0
	email := func(context.Context) error { enviados++; return nil }
	for range 3 {
		if err := d.Executar(t.Context(), "boas-vindas:c-42",
			email); err != nil {
			t.Fatal(err)
		}
	}
	if enviados != 1 {
		t.Fatalf("%d e-mails", enviados)
	}
}

func TestFalhaDoEfeitoLiberaAChave(t *testing.T) {
	d := &dedup.Dedup{DB: banco(t)}
	tentativas := 0
	email := func(context.Context) error {
		tentativas++
		if tentativas == 1 {
			return errors.New("SMTP recusou a conexão")
		}
		return nil
	}
	_ = d.Executar(t.Context(), "k", email)
	if err := d.Executar(t.Context(), "k", email); err != nil {
		t.Fatal(err)
	}
	if tentativas != 2 {
		t.Fatalf("%d tentativas", tentativas)
	}
}

// A queda entre o efeito e a marca: a próxima tentativa não sabe se o
// e-mail saiu, e diz isso.
func TestQuedaDepoisDoEfeito(t *testing.T) {
	d := &dedup.Dedup{DB: banco(t),
		Falha: func() error { return errors.New("processo morreu") }}
	enviados := 0
	email := func(context.Context) error { enviados++; return nil }
	_ = d.Executar(t.Context(), "k", email)
	d.Falha = nil
	err := d.Executar(t.Context(), "k", email)
	if !errors.Is(err, dedup.ErrIncerto) || enviados != 1 {
		t.Fatalf("%v, %d envios", err, enviados)
	}
}

func TestLocalExatamenteUmaVez(t *testing.T) {
	db := banco(t)
	d := &dedup.Dedup{DB: db}
	creditar := func(ctx context.Context, tx pgx.Tx) error {
		_, err := tx.Exec(ctx,
			`INSERT INTO ponto VALUES ('c-42', 100)`)
		return err
	}
	for range 3 {
		if err := d.ExecutarLocal(t.Context(), "pontos:pedido-7",
			creditar); err != nil {
			t.Fatal(err)
		}
	}
	var n int
	if err := db.QueryRow(t.Context(),
		`SELECT count(*) FROM ponto`).Scan(&n); err != nil || n != 1 {
		t.Fatalf("%d créditos, %v", n, err)
	}
}
