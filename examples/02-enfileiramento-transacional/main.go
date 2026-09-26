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
	"log/slog"
	"os"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/go-sob-pressao/enxame/internal/core/job"
	"github.com/go-sob-pressao/enxame/internal/store/postgres"
	"github.com/go-sob-pressao/enxame/internal/worker"
	"github.com/go-sob-pressao/enxame/internal/worker/runner"
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
	db, err := abrir(ctx, os.Getenv("ENXAME_DB_DSN"))
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

// trabalhar roda um worker até a fila da cobrança esvaziar.
func trabalhar(ctx context.Context, db *pgxpool.Pool) error {
	s := postgres.New(db)
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	p := &worker.Pool{
		Queue: postgres.NewFila(ctx, s), QueueName: "default",
		Concurrency: 2,
		Handlers: map[string]runner.Handler{
			"cobrar-pedido": func(_ context.Context, j job.Job) error {
				fmt.Printf("worker: cobrando %s\n", j.Args)
				return nil
			},
		},
		PollTimeout: time.Second, AttemptTimeout: 10 * time.Second,
		RetryDelay: time.Second, ReportEvery: time.Hour,
		Worker: "exemplo-02", Now: time.Now,
		Log: slog.New(slog.DiscardHandler),
	}
	go func() {
		for ctx.Err() == nil {
			var n int
			_ = db.QueryRow(ctx, `SELECT count(*) FROM job
				WHERE namespace = 'exemplo-02'
				  AND state <> 'completed'`).Scan(&n)
			if n == 0 {
				cancel()
			}
			<-time.After(100 * time.Millisecond)
		}
	}()
	if err := p.Run(ctx); !errors.Is(err, context.Canceled) {
		return err
	}
	return nil
}

// abrir recria o banco próprio do exemplo a cada execução.
func abrir(ctx context.Context, dsn string) (*pgxpool.Pool, error) {
	const banco = "enxame_exemplo02"
	admin, err := pgx.Connect(ctx, dsn)
	if err != nil {
		return nil, err
	}
	defer func() { _ = admin.Close(ctx) }()
	for _, sql := range []string{
		"DROP DATABASE IF EXISTS " + banco + " WITH (FORCE)",
		"CREATE DATABASE " + banco,
	} {
		if _, err := admin.Exec(ctx, sql); err != nil {
			return nil, err
		}
	}
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, err
	}
	cfg.ConnConfig.Database = banco
	db, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, err
	}
	if err := postgres.Migrate(ctx, db); err != nil {
		return nil, err
	}
	_, err = db.Exec(ctx, `
		CREATE TABLE pedido (
		    id INT PRIMARY KEY, quantidade INT NOT NULL);
		CREATE TABLE estoque (unidades INT NOT NULL);
		INSERT INTO estoque VALUES (10);`)
	return db, err
}
