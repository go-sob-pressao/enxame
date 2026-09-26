package dualwrite_test

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/go-sob-pressao/enxame/examples/cap15/dualwrite"
	"github.com/go-sob-pressao/enxame/test/testutil"
)

var errQueda = errors.New("processo morreu")

// cobranca é o consumidor da fila: recebe o id e procura o pedido.
type cobranca struct {
	db        *pgxpool.Pool
	cobrados  []int
	fantasmas []int // mensagens de pedidos que não estavam no banco
}

func (c *cobranca) Publicar(ctx context.Context, id int) error {
	var n int
	err := c.db.QueryRow(ctx,
		`SELECT count(*) FROM pedido WHERE id = $1`, id).Scan(&n)
	if err != nil {
		return err
	}
	if n == 0 {
		c.fantasmas = append(c.fantasmas, id)
		return nil
	}
	c.cobrados = append(c.cobrados, id)
	return nil
}

func banco(t *testing.T) *pgxpool.Pool {
	t.Helper()
	db := testutil.Postgres(t)
	if _, err := db.Exec(t.Context(), dualwrite.Esquema); err != nil {
		t.Fatal(err)
	}
	return db
}

func pedidos(t *testing.T, db *pgxpool.Pool) int {
	t.Helper()
	var n int
	err := db.QueryRow(t.Context(),
		`SELECT count(*) FROM pedido`).Scan(&n)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		t.Fatal(err)
	}
	return n
}
