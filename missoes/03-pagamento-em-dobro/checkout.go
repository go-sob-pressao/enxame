// Package pagamento é a Missão #3: o checkout que, às vezes, cobra o
// mesmo pedido duas vezes.
package pagamento

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/go-sob-pressao/enxame/pkg/enxame"
)

// Esquema cria as tabelas do checkout.
const Esquema = `
CREATE TABLE pedido (
    id    TEXT PRIMARY KEY,
    valor INT  NOT NULL
);
CREATE TABLE estoque (
    produto  TEXT PRIMARY KEY,
    unidades INT  NOT NULL CHECK (unidades >= 0)
);
INSERT INTO estoque VALUES ('camiseta', 100);`

// Cobrar são os argumentos do job que chama o gateway de pagamento.
type Cobrar struct {
	PedidoID string `json:"pedido_id"`
	Valor    int    `json:"valor"`
}

// Kind escolhe o handler.
func (Cobrar) Kind() string { return "cobrar" }

// Checkout finaliza pedidos. Reservar é a reserva de estoque, na
// transação do pedido; em produção, ela falha de vez em quando com um
// erro transitório (timeout de lock), e o cliente tenta de novo.
type Checkout struct {
	DB       *pgxpool.Pool
	Enxame   *enxame.Client
	Reservar func(ctx context.Context, tx pgx.Tx) error
}

// livro:inicio missao-03

// Finalizar registra o pedido, reserva o estoque e agenda a cobrança.
// Se devolver erro, o cliente chama de novo com o mesmo pedido.
func (c *Checkout) Finalizar(
	ctx context.Context,
	pedidoID string,
	valor int,
) error {
	if _, err := c.Enxame.Insert(ctx,
		Cobrar{PedidoID: pedidoID, Valor: valor}); err != nil {
		return err
	}
	return pgx.BeginFunc(ctx, c.DB, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, `INSERT INTO pedido (id, valor)
			VALUES ($1, $2)`, pedidoID, valor)
		if err != nil {
			return err
		}
		return c.Reservar(ctx, tx)
	})
}

// livro:fim missao-03
