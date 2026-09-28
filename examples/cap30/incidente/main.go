// Command incidente reproduz o incidente do Capítulo 30: a cobrança de
// um cliente falha para sempre, e a fila dele envelhece.
//
//	go run ./examples/cap30/incidente worker -dsn … \
//	    -otlp localhost:4318
//	go run ./examples/cap30/incidente carga -api http://localhost:8080
//
// O worker cobra os pedidos da fila "cobranca", um cliente por chave
// de ordem. O cartão do cliente-7 está vencido, e o handler devolve um
// erro comum — que o Enxame trata como passageiro e repete —, em vez de
// um erro permanente.
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"

	"github.com/go-sob-pressao/enxame/internal/observ/logging"
	"github.com/go-sob-pressao/enxame/internal/observ/tracing"
	"github.com/go-sob-pressao/enxame/pkg/enxame"
)

type cobranca struct {
	Cliente string `json:"cliente"`
	Pedido  int    `json:"pedido"`
}

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "uso: incidente worker|carga")
		os.Exit(2)
	}
	fs := flag.NewFlagSet(os.Args[1], flag.ExitOnError)
	dsn := fs.String("dsn", os.Getenv("ENXAME_DB_DSN"), "PostgreSQL")
	otlp := fs.String("otlp", "", "coletor OTLP/HTTP")
	api := fs.String("api", "http://127.0.0.1:8080", "API do enxamed")
	taxa := fs.Int("taxa", 5, "pedidos por segundo")
	_ = fs.Parse(os.Args[2:])
	ctx, parar := signal.NotifyContext(context.Background(),
		os.Interrupt, syscall.SIGTERM)
	defer parar()
	servico := map[string]string{"worker": "cobrador",
		"carga": "loja"}[os.Args[1]]
	desligar, err := tracing.Iniciar(ctx, tracing.Config{
		Servico: servico, Endpoint: *otlp, Amostragem: 1})
	if err == nil {
		defer func() { _ = desligar(context.WithoutCancel(ctx)) }()
		switch os.Args[1] {
		case "worker":
			err = trabalhar(ctx, *dsn)
		case "carga":
			err = carga(ctx, *api, *taxa)
		}
	}
	if err != nil && !errors.Is(err, context.Canceled) {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// livro:inicio handler-incidente

// cobrar chama o gateway de pagamento. O cartão vencido é um erro que
// nenhuma repetição resolve, e o handler o devolve como um erro comum.
func cobrar(log *slog.Logger) enxame.Handler {
	return func(ctx context.Context, j enxame.Job) error {
		var c cobranca
		if err := json.Unmarshal(j.Args, &c); err != nil {
			return enxame.Permanent(err)
		}
		<-time.After(20 * time.Millisecond) // o gateway
		if c.Cliente == "cliente-7" {
			err := errors.New("gateway recusou: cartão vencido")
			log.WarnContext(ctx, "cobrança recusada",
				slog.String("cliente", c.Cliente),
				slog.Int("pedido", c.Pedido), slog.Any("erro", err))
			return err // devia ser enxame.Permanent(err)
		}
		return nil
	}
}

// livro:fim handler-incidente

func trabalhar(ctx context.Context, dsn string) error {
	db, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return err
	}
	defer db.Close()
	log := slog.New(logging.Correlacao{
		Handler: slog.NewJSONHandler(os.Stderr, nil)})
	w := enxame.New(db, "loja").NewWorker(enxame.WorkerConfig{
		Queues: map[string]int{"cobranca": 8}, Log: log})
	w.Handle("cobrar", cobrar(log))
	return w.Run(ctx)
}

// carga cria pedidos de 20 clientes; cada pedido é um checkout, com o
// próprio trace, e a cobrança vai para a fila na ordem do cliente.
func carga(ctx context.Context, api string, taxa int) error {
	t := time.NewTicker(time.Second / time.Duration(taxa))
	defer t.Stop()
	for i := 0; ; i++ {
		c := cobranca{Cliente: fmt.Sprintf("cliente-%d", i%20),
			Pedido: i}
		corpo, _ := json.Marshal(map[string]any{"queue": "cobranca",
			"kind": "cobrar", "ordering_key": c.Cliente, "args": c})
		sctx, span := otel.Tracer("loja").Start(ctx, "checkout")
		req, err := http.NewRequestWithContext(sctx, http.MethodPost,
			api+"/v1/jobs", bytes.NewReader(corpo))
		if err != nil {
			span.End()
			return err
		}
		req.Header.Set("Authorization", "Bearer t1")
		otel.GetTextMapPropagator().Inject(sctx,
			propagation.HeaderCarrier(req.Header))
		if resp, err := http.DefaultClient.Do(req); err == nil {
			_ = resp.Body.Close()
		}
		span.End()
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-t.C:
		}
	}
}
