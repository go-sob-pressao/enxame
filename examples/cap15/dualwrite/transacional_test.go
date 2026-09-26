package dualwrite_test

import (
	"testing"

	"github.com/go-sob-pressao/enxame/examples/cap15/dualwrite"
	"github.com/go-sob-pressao/enxame/pkg/enxame"
)

// A mesma falha, nos mesmos pontos, contra a versão transacional: o
// pedido e o job existem juntos, ou nenhum dos dois.
func TestTransacionalTudoOuNada(t *testing.T) {
	db := banco(t)
	c := enxame.New(db, "loja")
	s := &dualwrite.Servico{DB: db,
		Falha: func() error { return errQueda }}
	_ = s.CriarTransacional(t.Context(), c, 1) // cai antes do COMMIT
	s.Falha = nil
	if err := s.CriarTransacional(t.Context(), c, 2); err != nil {
		t.Fatal(err)
	}
	var jobs int
	if err := db.QueryRow(t.Context(), `SELECT count(*) FROM job
		WHERE kind = 'cobrar-pedido'`).Scan(&jobs); err != nil {
		t.Fatal(err)
	}
	if n := pedidos(t, db); n != 1 || jobs != 1 {
		t.Fatalf("%d pedidos, %d jobs de cobrança", n, jobs)
	}
}
