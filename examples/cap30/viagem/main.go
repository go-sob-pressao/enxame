// Command viagem é um workflow de entrega cujo trace atravessa dois
// reinícios do worker (Capítulo 30).
//
//	go run ./examples/cap30/viagem iniciar -dsn … \
//	    -otlp localhost:4318
//	go run ./examples/cap30/viagem worker -dsn … -otlp localhost:4318
//
// iniciar abre o trace do pedido e começa o run; worker roda os passos.
// Entre um passo e o seguinte, o workflow dorme — e o worker pode ser
// parado e subido de novo quantas vezes for: o próximo passo continua
// o mesmo trace, porque o contexto está gravado no job.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.opentelemetry.io/otel"

	"github.com/go-sob-pressao/enxame/internal/observ/logging"
	"github.com/go-sob-pressao/enxame/internal/observ/tracing"
	"github.com/go-sob-pressao/enxame/pkg/enxame"
	"github.com/go-sob-pressao/enxame/pkg/workflow"
)

func main() {
	fs := flag.NewFlagSet(os.Args[1], flag.ExitOnError)
	dsn := fs.String("dsn", os.Getenv("ENXAME_DB_DSN"), "PostgreSQL")
	otlp := fs.String("otlp", "", "coletor OTLP/HTTP")
	espera := fs.Duration("espera", 20*time.Second, "entre os passos")
	_ = fs.Parse(os.Args[2:])
	ctx, parar := signal.NotifyContext(context.Background(),
		os.Interrupt, syscall.SIGTERM)
	defer parar()
	desligar, err := tracing.Iniciar(ctx, tracing.Config{
		Servico: "entregas", No: fmt.Sprintf("worker-%d", os.Getpid()),
		Endpoint: *otlp, Amostragem: 1})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer func() { _ = desligar(context.WithoutCancel(ctx)) }()
	db, err := pgxpool.New(ctx, *dsn)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer db.Close()
	if err := enxame.Migrate(ctx, db); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	c := enxame.New(db, "loja")
	switch os.Args[1] {
	case "iniciar":
		sctx, span := otel.Tracer("loja").Start(ctx, "pedido 42")
		_, err = c.StartWorkflow(sctx, "entrega", "pedido-42", nil)
		span.End()
	case "worker":
		log := slog.New(logging.Correlacao{
			Handler: slog.NewJSONHandler(os.Stderr, nil)})
		w := c.NewWorker(enxame.WorkerConfig{Log: log})
		w.Workflow("entrega", entrega(log, *espera))
		err = w.Run(ctx)
	}
	if err != nil && !errors.Is(err, context.Canceled) {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func entrega(log *slog.Logger, espera time.Duration) enxame.Workflow {
	return func(c *workflow.Context, _ json.RawMessage) (any, error) {
		for i, p := range []string{"separar", "faturar", "despachar"} {
			if i > 0 {
				if err := workflow.Sleep(c, "esperar-"+p,
					espera); err != nil {
					return nil, err
				}
			}
			if _, err := workflow.Step(c, p,
				func(ctx context.Context) (int, error) {
					log.InfoContext(ctx, "passo",
						slog.String("nome", p),
						slog.Int("pid", os.Getpid()))
					return os.Getpid(), nil
				}); err != nil {
				return nil, err
			}
		}
		return "entregue", nil
	}
}
