// Command sobrevive: o Experimento 14.1 — kill -9 no meio do trabalho
// e conferência do histórico.
//
//	export ENXAME_DB_DSN=postgres://…   # o de make up
//	go build -o sobrevive ./examples/cap14/sobrevive
//	./sobrevive enfileirar 5000
//	./sobrevive trabalhar &   # e, dois segundos depois:
//	kill -9 %1
//	./sobrevive conferir
//	./sobrevive trabalhar -resgatar
//
// O kill -9 não dá ao processo chance de nada: nenhum defer, nenhum
// cancelamento, nenhum log. O que sobrevive é o que o banco confirmou.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/go-sob-pressao/enxame/internal/core/id"
	"github.com/go-sob-pressao/enxame/internal/core/job"
	"github.com/go-sob-pressao/enxame/internal/store/postgres"
	"github.com/go-sob-pressao/enxame/internal/worker"
	"github.com/go-sob-pressao/enxame/internal/worker/runner"
)

const namespace = "cap14-sobrevive"

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr,
			"uso: sobrevive enfileirar N | trabalhar [-resgatar]"+
				" | conferir")
		os.Exit(2)
	}
	ctx := context.Background()
	db, err := abrir(ctx, os.Getenv("ENXAME_DB_DSN"))
	if err == nil {
		switch os.Args[1] {
		case "enfileirar":
			err = enfileirar(ctx, db, os.Args[2:])
		case "trabalhar":
			err = trabalhar(ctx, db, os.Args[2:])
		case "conferir":
			err = conferir(ctx, db)
		default:
			err = fmt.Errorf("comando desconhecido: %s", os.Args[1])
		}
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// abrir conecta ao banco próprio do experimento, criando-o e aplicando
// as migrações na primeira vez — Migrate só sabe partir do zero.
func abrir(ctx context.Context, dsn string) (*pgxpool.Pool, error) {
	const banco = "enxame_sobrevive"
	admin, err := pgx.Connect(ctx, dsn)
	if err != nil {
		return nil, err
	}
	defer func() { _ = admin.Close(ctx) }()
	var existe bool
	if err := admin.QueryRow(ctx, `SELECT EXISTS (SELECT 1
		FROM pg_database WHERE datname = $1)`, banco,
	).Scan(&existe); err != nil {
		return nil, err
	}
	if !existe {
		if _, err := admin.Exec(ctx,
			"CREATE DATABASE "+banco); err != nil {
			return nil, err
		}
	}
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, err
	}
	cfg.ConnConfig.Database = banco
	db, err := pgxpool.NewWithConfig(ctx, cfg)
	if err == nil && !existe {
		err = postgres.Migrate(ctx, db)
	}
	return db, err
}

func enfileirar(
	ctx context.Context,
	db *pgxpool.Pool,
	args []string,
) error {
	n, err := strconv.Atoi(args[0])
	if err != nil {
		return err
	}
	if _, err := db.Exec(ctx,
		`DELETE FROM job WHERE namespace = $1`, namespace); err != nil {
		return err
	}
	s := postgres.New(db)
	for range n {
		evs, j, err := job.Insert(job.Spec{
			ID: id.JobID(uuid.NewV7()), Namespace: namespace,
			Queue: namespace, Kind: "trabalho",
		}, time.Now())
		if err != nil {
			return err
		}
		if j, err = job.ApplyAll(j, evs); err != nil {
			return err
		}
		if err := s.Insert(ctx, j, evs); err != nil {
			return err
		}
	}
	fmt.Printf("%d jobs enfileirados\n", n)
	return nil
}

func trabalhar(
	ctx context.Context,
	db *pgxpool.Pool,
	args []string,
) error {
	fs := flag.NewFlagSet("trabalhar", flag.ExitOnError)
	resgatar := fs.Bool("resgatar", false,
		"resgata antes os jobs em running de um processo morto")
	if err := fs.Parse(args); err != nil {
		return err
	}
	s := postgres.New(db)
	if *resgatar {
		// Só é seguro porque sabemos que o único worker morreu.
		agora := time.Now()
		n, err := s.Rescue(ctx, agora, agora)
		if err != nil {
			return err
		}
		fmt.Printf("%d jobs resgatados\n", n)
	}
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	p := &worker.Pool{
		Queue: postgres.NewFila(ctx, s), QueueName: namespace,
		Concurrency: 8,
		Handlers: map[string]runner.Handler{
			"trabalho": func(ctx context.Context, _ job.Job) error {
				select {
				case <-time.After(10 * time.Millisecond):
					return nil
				case <-ctx.Done():
					return ctx.Err()
				}
			},
		},
		PollTimeout: time.Second, AttemptTimeout: 10 * time.Second,
		RetryDelay: time.Millisecond, ReportEvery: time.Hour,
		Worker: fmt.Sprintf("pid-%d", os.Getpid()), Now: time.Now,
		Log: slog.New(slog.DiscardHandler),
	}
	go func() { // encerra quando não houver mais nada a fazer
		for ctx.Err() == nil {
			var restantes int
			_ = db.QueryRow(ctx, `SELECT count(*) FROM job
				WHERE namespace = $1 AND state <> 'completed'`,
				namespace).Scan(&restantes)
			if restantes == 0 {
				cancel()
			}
			<-time.After(200 * time.Millisecond)
		}
	}()
	if err := p.Run(ctx); !errors.Is(err, context.Canceled) {
		return err
	}
	return nil
}

// livro:inicio conferir

// conferir compara, para cada job, a projeção com o estado reconstruído
// do histórico, e procura lacunas em seq.
func conferir(ctx context.Context, db *pgxpool.Pool) error {
	rows, err := db.Query(ctx, `SELECT state, count(*) FROM job
		WHERE namespace = $1 GROUP BY state ORDER BY state`, namespace)
	if err != nil {
		return err
	}
	for rows.Next() {
		var estado string
		var n int
		if err := rows.Scan(&estado, &n); err != nil {
			return err
		}
		fmt.Printf("%-10s %5d\n", estado, n)
	}
	var lacunas int
	if err := db.QueryRow(ctx, `SELECT count(*) FROM (
		SELECT e.job_id FROM job_event e JOIN job j USING (job_id)
		WHERE j.namespace = $1 GROUP BY e.job_id
		HAVING max(e.seq) <> count(*)) l`, namespace,
	).Scan(&lacunas); err != nil {
		return err
	}
	s := postgres.New(db)
	divergentes := 0
	ids, err := db.Query(ctx,
		`SELECT job_id::text FROM job WHERE namespace = $1`, namespace)
	if err != nil {
		return err
	}
	var todos []string
	for ids.Next() {
		var t string
		if err := ids.Scan(&t); err != nil {
			return err
		}
		todos = append(todos, t)
	}
	for _, t := range todos {
		jid, _ := id.ParseJobID(t)
		proj, _, err := s.Get(ctx, jid)
		if err != nil {
			return err
		}
		evs, err := s.History(ctx, jid)
		if err != nil {
			return err
		}
		rec, err := job.ApplyAll(job.Job{}, evs)
		if err != nil || rec.State != proj.State ||
			rec.Attempt != proj.Attempt {
			divergentes++
		}
	}
	fmt.Printf("jobs com lacuna em seq: %d\n", lacunas)
	fmt.Printf("projeção ≠ replay:      %d\n", divergentes)
	return nil
}

// livro:fim conferir
