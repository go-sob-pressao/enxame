// Command realidade: o Teste de Realidade #1 — kill -9 no processo com
// 500 jobs e 50 workflows em andamento. Só usa pkg/enxame.
//
//	realidade preparar     recria o banco, enfileira e inicia tudo
//	realidade trabalhar    roda o worker até não sobrar trabalho
//	realidade conferir     conta o que aconteceu
//
// Cada efeito — de um job ou de um passo — grava uma linha com a chave
// de idempotência (efeito) e uma linha por execução (execucao). Um
// efeito duplicado seria uma chave com duas linhas em efeito; a
// execução repetida aparece como mais linhas em execucao do que em
// efeito.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/go-sob-pressao/enxame/examples/internal/exemplo"
	"github.com/go-sob-pressao/enxame/pkg/enxame"
	"github.com/go-sob-pressao/enxame/pkg/job"
	"github.com/go-sob-pressao/enxame/pkg/workflow"
)

const (
	banco     = "enxame_realidade1"
	namespace = "realidade"
	jobs      = 500
	workflows = 50
)

// Tarefa são os argumentos dos jobs.
type Tarefa struct {
	N int `json:"n"`
}

// Kind escolhe o handler.
func (Tarefa) Kind() string { return "tarefa" }

func main() {
	ctx := context.Background()
	var err error
	switch arg(1) {
	case "preparar":
		err = preparar(ctx)
	case "trabalhar":
		err = trabalhar(ctx)
	case "conferir":
		err = conferir(ctx)
	default:
		err = errors.New("uso: realidade preparar|trabalhar|conferir")
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func arg(i int) string {
	if len(os.Args) > i {
		return os.Args[i]
	}
	return ""
}

func abrir(ctx context.Context) (*pgxpool.Pool, error) {
	cfg, err := pgxpool.ParseConfig(os.Getenv("ENXAME_DB_DSN"))
	if err != nil {
		return nil, err
	}
	cfg.ConnConfig.Database = banco
	return pgxpool.NewWithConfig(ctx, cfg)
}

func preparar(ctx context.Context) error {
	db, err := exemplo.Banco(ctx, banco)
	if err != nil {
		return err
	}
	defer db.Close()
	_, err = db.Exec(ctx, `
		CREATE TABLE efeito (chave TEXT PRIMARY KEY);
		CREATE TABLE execucao (chave TEXT NOT NULL,
		    em TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp());`)
	if err != nil {
		return err
	}
	c := enxame.New(db, namespace)
	for i := range jobs {
		if _, err := c.Insert(ctx, Tarefa{N: i}); err != nil {
			return err
		}
	}
	for i := range workflows {
		if _, err := c.StartWorkflow(ctx, "pedido",
			fmt.Sprintf("pedido-%d", i), nil); err != nil {
			return err
		}
	}
	fmt.Printf("%d jobs e %d workflows\n", jobs, workflows)
	return nil
}

// efeito é o efeito de um job ou passo: registra a execução e, na
// mesma transação, o efeito — uma vez por chave.
func efeito(ctx context.Context, db *pgxpool.Pool) error {
	chave := job.IdempotencyKey(ctx)
	select { // o trabalho "de verdade"
	case <-time.After(20 * time.Millisecond):
	case <-ctx.Done():
		return ctx.Err()
	}
	return pgx.BeginFunc(ctx, db, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx,
			`INSERT INTO execucao (chave) VALUES ($1)`, chave)
		if err != nil {
			return err
		}
		_, err = tx.Exec(ctx, `INSERT INTO efeito (chave) VALUES ($1)
			ON CONFLICT (chave) DO NOTHING`, chave)
		return err
	})
}

func trabalhar(ctx context.Context) error {
	db, err := abrir(ctx)
	if err != nil {
		return err
	}
	defer db.Close()
	// O contexto do passo vem do workflow.Context.
	//nolint:contextcheck
	passo := func(c *workflow.Context, nome string) error {
		_, err := workflow.Step(c, nome,
			func(ctx context.Context) (bool, error) {
				return true, efeito(ctx, db)
			})
		return err
	}
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	go exemplo.Esperar(ctx, db, cancel)
	w := enxame.New(db, namespace).NewWorker(enxame.WorkerConfig{
		RescueAfter: 3 * time.Second, RetryBase: 100 * time.Millisecond,
	})
	w.Handle("tarefa", func(ctx context.Context, _ enxame.Job) error {
		return efeito(ctx, db)
	})
	w.Workflow("pedido", func(c *workflow.Context,
		_ json.RawMessage) (any, error) {
		if err := passo(c, "cobrar"); err != nil {
			return nil, err
		}
		err := workflow.Sleep(c, "esperar", time.Second)
		if err != nil {
			return nil, err
		}
		return nil, passo(c, "emitir-nota")
	})
	return w.Run(ctx)
}

func conferir(ctx context.Context) error {
	db, err := abrir(ctx)
	if err != nil {
		return err
	}
	defer db.Close()
	consultas := []struct{ nome, sql string }{
		{"jobs de tarefa concluídos", `SELECT count(*) FROM job
			WHERE kind = 'tarefa' AND state = 'completed'`},
		{"jobs não concluídos", `SELECT count(*) FROM job
			WHERE state <> 'completed'`},
		{"workflows concluídos", `SELECT count(*) FROM workflow_run
			WHERE state = 'completed'`},
		{"efeitos (chaves distintas)", `SELECT count(*) FROM efeito`},
		{"execuções", `SELECT count(*) FROM execucao`},
		{"chaves executadas mais de uma vez", `SELECT count(*) FROM
			(SELECT chave FROM execucao GROUP BY chave
			 HAVING count(*) > 1) r`},
		{"tentativas resgatadas", `SELECT count(*) FROM job_event
			WHERE event_type = 10`},
	}
	for _, q := range consultas {
		var n int
		if err := db.QueryRow(ctx, q.sql).Scan(&n); err != nil {
			return err
		}
		fmt.Printf("%-34s %4d\n", q.nome, n)
	}
	return nil
}
