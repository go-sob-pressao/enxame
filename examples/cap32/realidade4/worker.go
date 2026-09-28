package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/go-sob-pressao/enxame/internal/core/job"
	"github.com/go-sob-pressao/enxame/internal/store/postgres"
	tgrpc "github.com/go-sob-pressao/enxame/internal/transport/grpc"
	"github.com/go-sob-pressao/enxame/internal/worker"
	"github.com/go-sob-pressao/enxame/internal/worker/poller"
	"github.com/go-sob-pressao/enxame/internal/worker/runner"
	wf "github.com/go-sob-pressao/enxame/internal/worker/workflow"
	"github.com/go-sob-pressao/enxame/pkg/workflow"
)

// preparar cria as tabelas dos efeitos.
func preparar(ctx context.Context, db *pgxpool.Pool) error {
	_, err := db.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS r4_efeito (
			n INT PRIMARY KEY, execucoes INT NOT NULL);
		CREATE TABLE IF NOT EXISTS r4_passo (
			n INT, passo TEXT, execucoes INT NOT NULL,
			PRIMARY KEY (n, passo))`)
	return err
}

// efeito conta uma execução: idempotente pela chave, e o que passar
// de uma é trabalho repetido, não efeito duplicado.
func efeito(ctx context.Context, db *pgxpool.Pool, n int,
	passo string) error {
	if passo == "" {
		_, err := db.Exec(ctx, `INSERT INTO r4_efeito VALUES ($1, 1)
			ON CONFLICT (n) DO UPDATE
			SET execucoes = r4_efeito.execucoes + 1`, n)
		return err
	}
	_, err := db.Exec(ctx, `INSERT INTO r4_passo VALUES ($1, $2, 1)
		ON CONFLICT (n, passo) DO UPDATE
		SET execucoes = r4_passo.execucoes + 1`, n, passo)
	return err
}

// pedido é o workflow: reservar, esperar 60 s, confirmar.
func pedido(db *pgxpool.Pool) wf.Func {
	return func(c *workflow.Context, in json.RawMessage) (any, error) {
		var p Pedido
		if err := json.Unmarshal(in, &p); err != nil {
			return nil, workflow.Permanent(err)
		}
		passo := func(nome string) func(context.Context) (int, error) {
			return func(ctx context.Context) (int, error) {
				return p.N, efeito(ctx, db, p.N, nome)
			}
		}
		if _, err := workflow.Step(c, "reservar",
			passo("reservar")); err != nil {
			return nil, err
		}
		if err := workflow.Sleep(c, "esperar",
			60*time.Second); err != nil {
			return nil, err
		}
		return workflow.Step(c, "confirmar", passo("confirmar"))
	}
}

// livro:inicio worker-todos

// trabalhar é o worker remoto, com um stream para cada enxamed: o
// enxamed só entrega jobs das partições que são dele, e um worker
// ligado a um nó só deixaria as partições dos outros sem ninguém.
// Os endereços são os nomes estáveis do StatefulSet — enxame-0,
// enxame-1… —, que atravessam o reinício de cada nó.
func trabalhar(ctx context.Context, db *pgxpool.Pool, enderecos,
	nome string, log *slog.Logger) error {
	if err := preparar(ctx, db); err != nil {
		return err
	}
	// Cada handler recebe o contexto da tentativa, do pool.
	hs := handlers(db) //nolint:contextcheck
	g, ctx := errgroup.WithContext(ctx)
	for addr := range strings.SplitSeq(enderecos, ",") {
		g.Go(func() error {
			return worker.Supervisionar(ctx, log, addr,
				func(ctx context.Context) error {
					return volta(ctx, addr, nome, hs, log)
				})
		})
	}
	return g.Wait()
}

// volta é uma conexão com um enxamed, do começo ao fim. O nome é
// estável, mas o IP não: o pod novo ganha outro, e uma conexão gRPC
// só volta a perguntar ao DNS depois de 30 s. Por isso cada volta abre
// uma conexão nova, que resolve o nome de novo.
func volta(ctx context.Context, addr, nome string,
	hs map[string]runner.Handler, log *slog.Logger) error {
	opts := append(tgrpc.Cliente("w", 5*time.Second),
		grpc.WithTransportCredentials(insecure.NewCredentials()))
	conn, err := grpc.NewClient(addr, opts...)
	if err != nil {
		return err
	}
	defer func() { _ = conn.Close() }()
	q, err := poller.Conectar(ctx, conn, nome, filas, 8)
	if err != nil {
		return fmt.Errorf("conectar a %s: %w", addr, err)
	}
	p := &worker.Pool{Queue: q, QueueName: "carga", Concurrency: 8,
		Handlers: hs, PollTimeout: time.Second,
		AttemptTimeout: time.Minute, RetryDelay: time.Second,
		ReportEvery: time.Hour, Heartbeat: q.Heartbeat,
		HeartbeatEvery: time.Second, Worker: nome, Now: time.Now,
		Log: log}
	return p.Run(ctx)
}

// livro:fim worker-todos

var filas = []string{"carga", workflow.AdvanceQueue}

// handlers são os jobs de meio segundo e os passos do workflow.
func handlers(db *pgxpool.Pool) map[string]runner.Handler {
	trabalho := func(ctx context.Context, j job.Job) error {
		var t Trabalho
		if err := json.Unmarshal(j.Args, &t); err != nil {
			return runner.Permanent(err)
		}
		select {
		case <-time.After(500 * time.Millisecond):
		case <-ctx.Done():
			return ctx.Err()
		}
		return efeito(ctx, db, t.N, "")
	}
	r := &wf.Replayer{Store: postgres.New(db).RelogioDoBanco(),
		Funcs: map[string]wf.Func{"pedido": pedido(db)},
		Now:   time.Now}
	return map[string]runner.Handler{"trabalho": trabalho,
		workflow.AdvanceKind: r.Handler()}
}
