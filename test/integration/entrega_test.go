//go:build integration

package integration_test

import (
	"context"
	"encoding/base64"
	"log/slog"
	"net/http"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/go-sob-pressao/enxame/internal/core/webhook"
	"github.com/go-sob-pressao/enxame/internal/delivery"
	"github.com/go-sob-pressao/enxame/internal/store/postgres"
	"github.com/go-sob-pressao/enxame/internal/transport/resilience"
	"github.com/go-sob-pressao/enxame/internal/worker"
	"github.com/go-sob-pressao/enxame/internal/worker/runner"
	"github.com/go-sob-pressao/enxame/pkg/enxame"
	pkgwebhook "github.com/go-sob-pressao/enxame/pkg/webhook"
	"github.com/go-sob-pressao/enxame/test/testutil"
)

var segredoTeste = "whsec_" + base64.StdEncoding.EncodeToString(
	[]byte("segredo de teste com 32 bytes!!!"))

type cenarioEntrega struct {
	db       *pgxpool.Pool
	s        *postgres.Store
	endpoint *testutil.Endpoint
	eid      string
	breakers *resilience.Breakers
}

// entrega sobe o endpoint, inscreve-o, publica uma mensagem e roda um
// pool de entrega até ctx acabar.
func entrega(
	t *testing.T,
	responder func(int) int,
	limiar int,
) *cenarioEntrega {
	t.Helper()
	t.Setenv("SEGREDO_TESTE", segredoTeste)
	c := &cenarioEntrega{db: testutil.Postgres(t)}
	c.s = postgres.New(c.db)
	c.endpoint = testutil.NovoEndpoint(t, segredoTeste, responder)
	e, err := c.s.CreateEndpoint(t.Context(), webhook.Endpoint{
		Namespace: "loja", URL: c.endpoint.URL,
		EventTypes: []string{"pedido.pago"},
		SecretRef:  "env:SEGREDO_TESTE"})
	if err != nil {
		t.Fatal(err)
	}
	c.eid = e.ID
	cli := enxame.New(c.db, "loja")
	if err := pgx.BeginFunc(t.Context(), c.db, func(tx pgx.Tx) error {
		_, err := cli.PublishTx(t.Context(), tx, pkgwebhook.Message{
			EventType: "pedido.pago", Payload: map[string]int{"id": 42}})
		return err
	}); err != nil {
		t.Fatal(err)
	}
	c.breakers = &resilience.Breakers{Limiar: limiar, Pausa: time.Hour}
	d := &delivery.Entregador{Store: c.s, Cliente: delivery.NovoCliente(),
		Breakers: c.breakers, LimitePorEndpoint: 4,
		Segredo: delivery.SegredoDoAmbiente, Now: time.Now}
	ctx, cancel := context.WithCancel(t.Context())
	t.Cleanup(cancel)
	p := &worker.Pool{Queue: postgres.NewFila(ctx, c.s),
		QueueName: pkgwebhook.FanoutQueue, Concurrency: 2,
		Handlers: map[string]runner.Handler{
			pkgwebhook.FanoutKind:  d.Fanout,
			pkgwebhook.DeliverKind: d.Entregar},
		PollTimeout: time.Second, AttemptTimeout: 10 * time.Second,
		RetryDelay: 20 * time.Millisecond, ReportEvery: time.Hour,
		Worker: "entrega", Now: time.Now,
		Log: slog.New(slog.DiscardHandler)}
	go func() { _ = p.Run(ctx) }()
	return c
}

func (c *cenarioEntrega) esperar(t *testing.T, sql string, n int) {
	t.Helper()
	for range 1000 {
		if contar(t, c.db, sql) >= n {
			return
		}
		<-time.After(20 * time.Millisecond)
	}
	t.Fatalf("%s: não chegou a %d", sql, n)
}

// livro:inicio entrega-teste

// O endpoint responde 500 três vezes e depois 200. A entrega termina,
// cada tentativa está registrada, e as quatro requisições trazem o
// mesmo webhook-id — o que permite ao cliente deduplicar.
func TestEntregaComFalhasPassageiras(t *testing.T) {
	c := entrega(t, func(n int) int {
		if n <= 3 {
			return http.StatusInternalServerError
		}
		return http.StatusOK
	}, 10)
	c.esperar(t, `SELECT count(*) FROM job
		WHERE kind = 'webhook.deliver' AND state = 'completed'`, 1)
	recebidas, validas, ids := c.endpoint.Contagem()
	tentativas := contar(t, c.db,
		`SELECT count(*) FROM webhook_attempt`)
	t.Logf("requisições %d, válidas %d, webhook-id distintos %d, "+
		"tentativas registradas %d", recebidas, validas, ids,
		tentativas)
	if recebidas != 4 || validas != 4 || ids != 1 || tentativas != 4 {
		t.Fatal("contagem inesperada")
	}
}

// Sempre 500: depois de três falhas seguidas, o breaker abre, e as
// tentativas seguintes nem chegam ao endpoint.
func TestBreakerPoupaOEndpoint(t *testing.T) {
	c := entrega(t, func(int) int {
		return http.StatusInternalServerError
	}, 3)
	c.esperar(t, `SELECT count(*) FROM job WHERE kind =
		'webhook.deliver' AND attempt >= 6`, 1)
	recebidas, _, _ := c.endpoint.Contagem()
	t.Logf("6 tentativas do job; %d chegaram ao endpoint; breaker %v",
		recebidas, c.breakers.Estado(c.eid, time.Now()))
	if recebidas != 3 {
		t.Fatalf("%d requisições com o breaker aberto", recebidas)
	}
}

// livro:fim entrega-teste

// 410 Gone: o endpoint é desativado, e a entrega, descartada.
func TestGoneDesativaOEndpoint(t *testing.T) {
	c := entrega(t, func(int) int { return http.StatusGone }, 10)
	c.esperar(t, `SELECT count(*) FROM job WHERE kind =
		'webhook.deliver' AND state = 'discarded'`, 1)
	e, err := c.s.Endpoint(t.Context(), c.eid)
	if err != nil || !e.Disabled {
		t.Fatalf("endpoint %+v, %v", e, err)
	}
}

// O endpoint responde 429 com Retry-After: 2 na primeira; a segunda
// tentativa só sai depois de dois segundos, apesar do backoff de 20 ms.
func TestRetryAfterRespeitado(t *testing.T) {
	var primeira, segunda time.Time
	c := entrega(t, func(n int) int {
		if n == 1 {
			primeira = time.Now()
			return http.StatusTooManyRequests
		}
		segunda = time.Now()
		return http.StatusOK
	}, 10)
	c.esperar(t, `SELECT count(*) FROM job
		WHERE kind = 'webhook.deliver' AND state = 'completed'`, 1)
	t.Logf("segunda tentativa %v depois da primeira",
		segunda.Sub(primeira).Round(100*time.Millisecond))
	if segunda.Sub(primeira) < 2*time.Second {
		t.Fatal("o Retry-After não foi respeitado")
	}
}
