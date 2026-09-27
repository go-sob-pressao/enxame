// Command cluster é o Teste de Realidade #3 em peças: workers remotos
// que gravam o efeito de cada job, um gerador de carga pela API e a
// conferência no fim (Capítulo 26).
//
//	go run ./examples/cap26/cluster preparar -dsn …
//	go run ./examples/cap26/cluster worker -dsn … -grpc :7281 -nome w1
//	go run ./examples/cap26/cluster carga -api http://localhost:8181 \
//	    -n 600 -taxa 100
//	go run ./examples/cap26/cluster conferir -dsn … -n 600
//
// Cada job leva uma chave de ordem (chave-0 a chave-19) e um número de
// sequência dentro dela. O worker grava, para cada job, quantas vezes o
// efeito rodou e, para cada chave, a ordem em que os números chegaram.
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/go-sob-pressao/enxame/internal/core/job"
	tgrpc "github.com/go-sob-pressao/enxame/internal/transport/grpc"
	"github.com/go-sob-pressao/enxame/internal/worker"
	"github.com/go-sob-pressao/enxame/internal/worker/poller"
	"github.com/go-sob-pressao/enxame/internal/worker/runner"
)

const chaves = 20

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr,
			"uso: cluster preparar|worker|carga|conferir")
		os.Exit(2)
	}
	fs := flag.NewFlagSet(os.Args[1], flag.ExitOnError)
	dsn := fs.String("dsn", os.Getenv("ENXAME_DB_DSN"), "PostgreSQL")
	addr := fs.String("grpc", "127.0.0.1:7281", "gRPC do enxamed")
	nome := fs.String("nome", "w1", "nome do worker")
	api := fs.String("api", "http://127.0.0.1:8181", "API do enxamed")
	n := fs.Int("n", 600, "jobs")
	taxa := fs.Int("taxa", 100, "jobs por segundo")
	_ = fs.Parse(os.Args[2:])
	ctx, parar := signal.NotifyContext(context.Background(),
		os.Interrupt, syscall.SIGTERM)
	defer parar()
	var err error
	switch os.Args[1] {
	case "preparar":
		err = comBanco(ctx, *dsn, preparar)
	case "worker":
		err = comBanco(ctx, *dsn, func(ctx context.Context,
			db *pgxpool.Pool) error {
			return trabalhar(ctx, db, *addr, *nome)
		})
	case "carga":
		err = carga(ctx, *api, *n, *taxa)
	case "conferir":
		err = comBanco(ctx, *dsn, func(ctx context.Context,
			db *pgxpool.Pool) error {
			return conferir(ctx, db, *n)
		})
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func comBanco(ctx context.Context, dsn string,
	f func(context.Context, *pgxpool.Pool) error) error {
	db, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return err
	}
	defer db.Close()
	return f(ctx, db)
}

func preparar(ctx context.Context, db *pgxpool.Pool) error {
	_, err := db.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS efeito (
			n INT PRIMARY KEY, execucoes INT NOT NULL);
		CREATE TABLE IF NOT EXISTS chegada (
			id BIGSERIAL PRIMARY KEY, chave INT, seq INT)`)
	return err
}

type trabalho struct {
	N     int `json:"n"`     // número do job, de 0 a n-1
	Chave int `json:"chave"` // chave de ordem
	Seq   int `json:"seq"`   // ordem dentro da chave
}

// livro:inicio worker-cluster

// trabalhar conecta um worker remoto ao enxamed e roda jobs de 50 ms.
// O efeito é idempotente pela chave — o número do job —, e conta as
// execuções: o que passar de uma é trabalho repetido, não efeito
// duplicado.
func trabalhar(ctx context.Context, db *pgxpool.Pool, addr,
	nome string) error {
	opts := append(tgrpc.Cliente("w", 5*time.Second),
		grpc.WithTransportCredentials(insecure.NewCredentials()))
	conn, err := grpc.NewClient(addr, opts...)
	if err != nil {
		return err
	}
	defer func() { _ = conn.Close() }()
	r, err := poller.Conectar(ctx, conn, nome, []string{"carga"}, 4)
	if err != nil {
		return err
	}
	h := func(ctx context.Context, j job.Job) error {
		var t trabalho
		if err := json.Unmarshal(j.Args, &t); err != nil {
			return runner.Permanent(err)
		}
		select {
		case <-time.After(50 * time.Millisecond):
		case <-ctx.Done():
			return ctx.Err()
		}
		return pgx.BeginFunc(ctx, db, func(tx pgx.Tx) error {
			_, err := tx.Exec(ctx, `INSERT INTO efeito VALUES ($1, 1)
				ON CONFLICT (n)
				DO UPDATE SET execucoes = efeito.execucoes + 1`, t.N)
			if err != nil {
				return err
			}
			_, err = tx.Exec(ctx, `INSERT INTO chegada (chave, seq)
				VALUES ($1, $2)`, t.Chave, t.Seq)
			return err
		})
	}
	p := &worker.Pool{Queue: r, QueueName: "carga", Concurrency: 4,
		Handlers:    map[string]runner.Handler{"trabalho": h},
		PollTimeout: time.Second, AttemptTimeout: time.Minute,
		RetryDelay: time.Second, ReportEvery: time.Hour,
		Heartbeat: r.Heartbeat, HeartbeatEvery: time.Second,
		Worker: nome, Now: time.Now, Log: slog.New(slog.DiscardHandler)}
	return p.Run(ctx)
}

// livro:fim worker-cluster

// carga enfileira n jobs pela API, à taxa dada, com chaves de ordem.
func carga(ctx context.Context, api string, n, taxa int) error {
	passo := time.Second / time.Duration(taxa)
	seq := make([]int, chaves)
	for i := range n {
		k := i % chaves
		corpo, _ := json.Marshal(map[string]any{
			"queue": "carga", "kind": "trabalho",
			"ordering_key": fmt.Sprintf("chave-%d", k),
			"args":         trabalho{N: i, Chave: k, Seq: seq[k]}})
		seq[k]++
		req, err := http.NewRequestWithContext(ctx, http.MethodPost,
			api+"/v1/jobs", bytes.NewReader(corpo))
		if err != nil {
			return err
		}
		req.Header.Set("Authorization", "Bearer t1")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return err
		}
		_ = resp.Body.Close()
		if resp.StatusCode != http.StatusCreated {
			return fmt.Errorf("job %d: %s", i, resp.Status)
		}
		time.Sleep(passo)
	}
	return nil
}

// conferir diz quantos jobs rodaram, quantos rodaram mais de uma vez e
// se a ordem de cada chave foi respeitada.
func conferir(ctx context.Context, db *pgxpool.Pool, n int) error {
	var feitos, repetidos, pendentes, foraDeOrdem int
	_ = db.QueryRow(ctx, `SELECT count(*), count(*) FILTER
		(WHERE execucoes > 1) FROM efeito`).Scan(&feitos, &repetidos)
	_ = db.QueryRow(ctx, `SELECT count(*) FROM job WHERE queue = 'carga'
		AND state <> 'completed'`).Scan(&pendentes)
	// Fora de ordem: uma chegada com seq menor que a anterior da chave,
	// descontadas as repetições de um mesmo seq.
	_ = db.QueryRow(ctx, `SELECT count(*) FROM (
		SELECT seq, max(seq) OVER (PARTITION BY chave ORDER BY id
		  ROWS BETWEEN UNBOUNDED PRECEDING AND 1 PRECEDING) AS antes
		FROM chegada) c WHERE seq < antes`).Scan(&foraDeOrdem)
	fmt.Printf("jobs: %d; com efeito: %d; perdidos: %d; "+
		"rodados mais de uma vez: %d; fora de ordem: %d\n",
		n, feitos, n-feitos, repetidos, foraDeOrdem)
	fmt.Printf("jobs ainda não concluídos: %d\n", pendentes)
	return nil
}
