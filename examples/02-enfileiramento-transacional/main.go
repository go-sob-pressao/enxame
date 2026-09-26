// Command enfileiramento-transacional: o pedido e o job de cobrança no
// mesmo COMMIT (Capítulo 15).
//
//	make up
//	export ENXAME_DB_DSN=postgres://…   # o de make up
//	go run ./examples/02-enfileiramento-transacional
//
// Cria dois pedidos. O segundo falha na checagem de estoque depois de
// enfileirar a cobrança; o ROLLBACK leva o pedido e a cobrança juntos.
// Depois, um worker processa o que ficou na fila.
package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strconv"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/go-sob-pressao/enxame/examples/internal/exemplo"
	"github.com/go-sob-pressao/enxame/pkg/enxame"
)

// CobrarPedido são os argumentos do job de cobrança.
type CobrarPedido struct {
	PedidoID int `json:"pedido_id"`
}

// Kind escolhe o handler.
func (CobrarPedido) Kind() string { return "cobrar-pedido" }

var errEstoque = errors.New("estoque insuficiente")

// livro:inicio exemplo-02

// criarPedido grava o pedido e enfileira a cobrança na mesma transação.
// A checagem de estoque vem depois do enfileiramento de propósito: se
// ela falhar, o ROLLBACK desfaz os dois.
func criarPedido(
	ctx context.Context,
	db *pgxpool.Pool,
	c *enxame.Client,
	id, quantidade int,
) error {
	return pgx.BeginFunc(ctx, db, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, `INSERT INTO pedido (id, quantidade)
			VALUES ($1, $2)`, id, quantidade)
		if err != nil {
			return err
		}
		_, err = c.InsertTx(ctx, tx, CobrarPedido{PedidoID: id},
			enxame.UniqueKey("cobranca:"+strconv.Itoa(id)))
		if err != nil {
			return err
		}
		var estoque int
		if err := tx.QueryRow(ctx,
			`SELECT unidades FROM estoque`).Scan(&estoque); err != nil {
			return err
		}
		if quantidade > estoque {
			return errEstoque // ROLLBACK: nem pedido, nem cobrança
		}
		_, err = tx.Exec(ctx,
			`UPDATE estoque SET unidades = unidades - $1`, quantidade)
		return err
	})
}

// livro:fim exemplo-02

func main() {
	ctx := context.Background()
	db, err := abrir(ctx)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer db.Close()
	c := enxame.New(db, "exemplo-02")
	for _, p := range []struct{ id, qtd int }{{1, 3}, {2, 50}} {
		err := criarPedido(ctx, db, c, p.id, p.qtd)
		fmt.Printf("pedido %d (%d unidades): %v\n", p.id, p.qtd,
			resultado(err))
	}
	if err := trabalhar(ctx, db); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func resultado(err error) string {
	if err != nil {
		return "desfeito — " + err.Error()
	}
	return "gravado, com a cobrança"
}

// trabalhar roda o worker embutido até a fila esvaziar.
func trabalhar(ctx context.Context, db *pgxpool.Pool) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	go exemplo.Esperar(ctx, db, cancel)
	w := enxame.New(db, "exemplo-02").NewWorker(enxame.WorkerConfig{})
	w.Handle("cobrar-pedido", func(_ context.Context,
		j enxame.Job) error {
		fmt.Printf("worker: cobrando %s\n", j.Args)
		return nil
	})
	return w.Run(ctx)
}

// abrir recria o banco do exemplo e cria as tabelas da aplicação.
func abrir(ctx context.Context) (*pgxpool.Pool, error) {
	db, err := exemplo.Banco(ctx, "enxame_exemplo02")
	if err != nil {
		return nil, err
	}
	_, err = db.Exec(ctx, `
		CREATE TABLE pedido (
		    id INT PRIMARY KEY, quantidade INT NOT NULL);
		CREATE TABLE estoque (unidades INT NOT NULL);
		INSERT INTO estoque VALUES (10);`)
	return db, err
}
