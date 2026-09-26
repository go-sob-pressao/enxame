package dualwrite_test

import (
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/go-sob-pressao/enxame/examples/cap15/dualwrite"
	"github.com/go-sob-pressao/enxame/test/testutil"
)

var errQueda = errors.New("processo morreu")

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
