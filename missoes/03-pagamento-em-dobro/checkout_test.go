package pagamento_test

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/go-sob-pressao/enxame/internal/core/job"
	"github.com/go-sob-pressao/enxame/internal/store/postgres"
	"github.com/go-sob-pressao/enxame/internal/worker"
	"github.com/go-sob-pressao/enxame/internal/worker/runner"
	pagamento "github.com/go-sob-pressao/enxame/missoes/03-pagamento-em-dobro"
	"github.com/go-sob-pressao/enxame/pkg/enxame"
	"github.com/go-sob-pressao/enxame/test/testutil"
)

// gateway é o provedor de pagamento: conta quantas vezes cada pedido
// foi cobrado.
type gateway struct {
	mu        sync.Mutex
	cobrancas map[string]int
}

func (g *gateway) cobrar(_ context.Context, j job.Job) error {
	var c pagamento.Cobrar
	if err := json.Unmarshal(j.Args, &c); err != nil {
		return runner.Permanent(err)
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	g.cobrancas[c.PedidoID]++
	return nil
}

var errLock = errors.New("lock timeout na reserva de estoque")

func preparar(t *testing.T) (*pgxpool.Pool, *pagamento.Checkout) {
	t.Helper()
	db := testutil.Postgres(t)
	if _, err := db.Exec(t.Context(), pagamento.Esquema); err != nil {
		t.Fatal(err)
	}
	tentativas := 0
	return db, &pagamento.Checkout{
		DB: db, Enxame: enxame.New(db, "loja"),
		// A primeira reserva falha com um erro transitório; as
		// seguintes funcionam.
		Reservar: func(ctx context.Context, tx pgx.Tx) error {
			tentativas++
			if tentativas == 1 {
				return errLock
			}
			_, err := tx.Exec(ctx, `UPDATE estoque
				SET unidades = unidades - 1 WHERE produto = 'camiseta'`)
			return err
		},
	}
}

// processar roda um worker até não haver mais cobrança pendente.
func processar(t *testing.T, db *pgxpool.Pool, g *gateway) {
	t.Helper()
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	p := &worker.Pool{
		Queue:     postgres.NewFila(ctx, postgres.New(db)),
		QueueName: "default", Concurrency: 2,
		Handlers:    map[string]runner.Handler{"cobrar": g.cobrar},
		PollTimeout: time.Second, AttemptTimeout: 10 * time.Second,
		RetryDelay: time.Millisecond, ReportEvery: time.Hour,
		Worker: "w", Now: time.Now, Log: slog.New(slog.DiscardHandler),
	}
	fim := make(chan error, 1)
	go func() { fim <- p.Run(ctx) }()
	for range 200 {
		var n int
		if err := db.QueryRow(t.Context(), `SELECT count(*) FROM job
			WHERE state <> 'completed'`).Scan(&n); err != nil {
			t.Fatal(err)
		}
		if n == 0 {
			return
		}
		<-time.After(50 * time.Millisecond)
	}
	t.Fatal("cobranças pendentes depois de 10 s")
}

// livro:inicio missao-03-teste

// O cliente finaliza o pedido P-1 e, diante de um erro, tenta de novo
// até três vezes. O pedido precisa estar gravado e ter sido cobrado
// exatamente uma vez.
func TestMissao(t *testing.T) {
	db, checkout := preparar(t)
	var err error
	for range 3 {
		err = checkout.Finalizar(t.Context(), "P-1", 100)
		if err == nil {
			break
		}
	}
	if err != nil {
		t.Fatalf("o cliente desistiu: %v", err)
	}
	g := &gateway{cobrancas: map[string]int{}}
	processar(t, db, g)
	var pedidos int
	err = db.QueryRow(t.Context(),
		`SELECT count(*) FROM pedido WHERE id = 'P-1'`).Scan(&pedidos)
	if err != nil {
		t.Fatal(err)
	}
	if pedidos != 1 || g.cobrancas["P-1"] != 1 {
		t.Fatalf("pedido P-1: gravado %d vez(es), cobrado %d vez(es)",
			pedidos, g.cobrancas["P-1"])
	}
}

// livro:fim missao-03-teste
