package http_test

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	api "github.com/go-sob-pressao/enxame/internal/transport/http"
)

var ok = http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusNoContent)
})

func pedir(
	t *testing.T,
	h http.Handler,
	token string,
) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet,
		"/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

// Cada namespace tem o seu balde: esgotar o de um não afeta o outro.
func TestLimitarPorNamespace(t *testing.T) {
	a := api.NovaAPI(nil, map[string]string{"ta": "a", "tb": "b"},
		slog.New(slog.DiscardHandler))
	h := api.Encadear(ok, a.Autenticar, api.Limitar(1, 3))
	for i := range 3 {
		if r := pedir(t, h, "ta"); r.Code != http.StatusNoContent {
			t.Fatalf("pedido %d de a: %d", i+1, r.Code)
		}
	}
	r := pedir(t, h, "ta")
	if r.Code != http.StatusTooManyRequests ||
		r.Header().Get("Retry-After") != "1" {
		t.Fatalf("quarto pedido de a: %d, Retry-After %q",
			r.Code, r.Header().Get("Retry-After"))
	}
	if r := pedir(t, h, "tb"); r.Code != http.StatusNoContent {
		t.Fatalf("b pagou pelo excesso de a: %d", r.Code)
	}
}

// Com as vagas ocupadas, a próxima requisição é recusada na hora — não
// espera uma vaga.
func TestDescartarRecusaSemEsperar(t *testing.T) {
	entrou, solta := make(chan struct{}), make(chan struct{})
	lento := http.HandlerFunc(func(w http.ResponseWriter,
		_ *http.Request) {
		entrou <- struct{}{}
		<-solta
		w.WriteHeader(http.StatusNoContent)
	})
	h := api.Descartar(2)(lento)
	var wg sync.WaitGroup
	for range 2 {
		wg.Go(func() { pedir(t, h, "") })
		<-entrou
	}
	r := pedir(t, h, "")
	if r.Code != http.StatusServiceUnavailable ||
		r.Header().Get("Retry-After") == "" {
		t.Fatalf("terceira em curso: %d", r.Code)
	}
	close(solta)
	wg.Wait()
	// As vagas voltam quando as requisições terminam.
	go func() { <-entrou }()
	if r := pedir(t, h, ""); r.Code != http.StatusNoContent {
		t.Fatalf("vaga livre recusada: %d", r.Code)
	}
}
