// Package dualwrite mostra como gravar o pedido no banco e avisar a
// cobrança por uma fila — três formas de fazer, duas delas erradas.
package dualwrite

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/go-sob-pressao/enxame/pkg/enxame"
)

// Esquema cria a tabela do exemplo.
const Esquema = `CREATE TABLE pedido (
    id    INT PRIMARY KEY,
    total INT NOT NULL
);`

// Publicador é a fila externa: um broker qualquer, fora do banco.
type Publicador interface {
	Publicar(ctx context.Context, pedidoID int) error
}

// Servico cria pedidos. Falha, quando não é nil, é chamada no ponto
// marcado de cada versão: é onde o teste injeta a queda do processo.
type Servico struct {
	DB    *pgxpool.Pool
	Fila  Publicador
	Falha func() error
}

func (s *Servico) falha() error {
	if s.Falha == nil {
		return nil
	}
	return s.Falha()
}

// livro:inicio dual-write

// CriarDualWrite grava o pedido, faz COMMIT e publica. Parece seguro:
// só publica o que já está gravado. Entre o COMMIT e a publicação, o
// processo pode morrer — e o pedido fica gravado sem cobrança.
func (s *Servico) CriarDualWrite(ctx context.Context, id int) error {
	_, err := s.DB.Exec(ctx,
		`INSERT INTO pedido (id, total) VALUES ($1, 100)`, id)
	if err != nil {
		return err
	}
	if err := s.falha(); err != nil { // a janela
		return err
	}
	return s.Fila.Publicar(ctx, id)
}

// livro:fim dual-write

// livro:inicio publica-antes

// CriarPublicaAntes grava, publica e comita — nessa ordem. Revisada por
// três pessoas: "se o COMMIT falhar, o ROLLBACK desfaz tudo".
func (s *Servico) CriarPublicaAntes(ctx context.Context, id int) error {
	return pgx.BeginFunc(ctx, s.DB, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx,
			`INSERT INTO pedido (id, total) VALUES ($1, 100)`, id)
		if err != nil {
			return err
		}
		if err := s.Fila.Publicar(ctx, id); err != nil {
			return err
		}
		return s.falha() // antes do COMMIT
	})
}

// livro:fim publica-antes

type cobrar struct {
	PedidoID int `json:"pedido_id"`
}

func (cobrar) Kind() string { return "cobrar-pedido" }

// livro:inicio transacional

// CriarTransacional grava o pedido e o job de cobrança na mesma
// transação. Não há ordem entre os dois: o COMMIT grava os dois, e
// qualquer falha antes dele não grava nenhum.
func (s *Servico) CriarTransacional(
	ctx context.Context,
	c *enxame.Client,
	id int,
) error {
	return pgx.BeginFunc(ctx, s.DB, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx,
			`INSERT INTO pedido (id, total) VALUES ($1, 100)`, id)
		if err != nil {
			return err
		}
		if _, err := c.InsertTx(ctx, tx, cobrar{id}); err != nil {
			return err
		}
		return s.falha() // antes do COMMIT
	})
}

// livro:fim transacional
