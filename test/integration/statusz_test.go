//go:build integration

package integration_test

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	api "github.com/go-sob-pressao/enxame/internal/transport/http"
)

// O /statusz diz o que o nó vê do banco, sem token — e diz 503 quando
// o banco não responde. O /healthz, no mesmo nó, continua 200: a
// liveness não pode depender do banco (Cap. 32).
func TestStatusz(t *testing.T) {
	a, _, c, _ := apiDeTeste(t)
	a.Particoes = func() int { return 171 }
	c.token = ""
	status, corpo := c.chamar(http.MethodGet, "/statusz", "")
	if status != http.StatusOK ||
		!strings.Contains(corpo, `"banco":"ok"`) ||
		!strings.Contains(corpo, `"particoes":171`) {
		t.Fatalf("banco no ar: %d %s", status, corpo)
	}

	// Um banco que não aceita conexões: a porta 1 do próprio host.
	fora, err := pgxpool.New(t.Context(),
		"postgres://x@127.0.0.1:1/x?connect_timeout=1")
	if err != nil {
		t.Fatal(err)
	}
	defer fora.Close()
	b := api.NovaAPI(fora, nil, slog.New(slog.DiscardHandler))
	srv := httptest.NewServer(b.Handler())
	defer srv.Close()
	c.base = srv.URL
	if status, corpo = c.chamar(http.MethodGet, "/statusz",
		""); status != http.StatusServiceUnavailable {
		t.Fatalf("banco fora: %d %s", status, corpo)
	}
	if status, _ = c.chamar(http.MethodGet, "/healthz",
		""); status != http.StatusOK {
		t.Fatalf("healthz com o banco fora: %d", status)
	}
}
