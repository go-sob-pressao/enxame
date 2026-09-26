//go:build defeito

package dualwrite_test

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/go-sob-pressao/enxame/examples/cap15/dualwrite"
)

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

// livro:inicio dual-write-teste

// O processo morre na janela entre o COMMIT e a publicação.
func TestDualWritePerdeCobranca(t *testing.T) {
	db := banco(t)
	fila := &cobranca{db: db}
	s := &dualwrite.Servico{DB: db, Fila: fila,
		Falha: func() error { return errQueda }}
	_ = s.CriarDualWrite(t.Context(), 1)
	if n := pedidos(t, db); n != len(fila.cobrados) {
		t.Fatalf("%d pedido gravado, %d cobrança", n,
			len(fila.cobrados))
	}
}

// livro:fim dual-write-teste

// livro:inicio publica-antes-teste

// Sem falha nenhuma: o consumidor recebe a mensagem antes do COMMIT.
func TestPublicaAntesSemFalha(t *testing.T) {
	db := banco(t)
	fila := &cobranca{db: db}
	s := &dualwrite.Servico{DB: db, Fila: fila}
	if err := s.CriarPublicaAntes(t.Context(), 1); err != nil {
		t.Fatal(err)
	}
	if len(fila.fantasmas) > 0 {
		t.Fatalf("cobrança recebeu pedido %v, que não achou no banco",
			fila.fantasmas)
	}
}

// O processo morre depois de publicar, antes do COMMIT.
func TestPublicaAntesComFalha(t *testing.T) {
	db := banco(t)
	fila := &cobranca{db: db}
	s := &dualwrite.Servico{DB: db, Fila: fila,
		Falha: func() error { return errQueda }}
	_ = s.CriarPublicaAntes(t.Context(), 1)
	if n := pedidos(t, db); n == 0 && len(fila.fantasmas) > 0 {
		t.Fatalf("mensagem do pedido %v publicada; pedidos no banco: 0",
			fila.fantasmas)
	}
}

// livro:fim publica-antes-teste
