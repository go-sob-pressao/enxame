package http

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
)

// O limite de requisições em curso é um só para todas as rotas, e o
// long-poll não ocupa vaga.
func TestProtecaoCompartilhada(t *testing.T) {
	a := NovaAPI(nil, map[string]string{"t": "ns"},
		slog.New(slog.DiscardHandler))
	a.MaxEmCurso = 1
	protecao := a.protecao()
	entrou, solta := make(chan struct{}), make(chan struct{})
	lenta := Encadear(http.HandlerFunc(func(_ http.ResponseWriter,
		_ *http.Request) {
		close(entrou)
		<-solta
	}), protecao...)
	outra := Encadear(http.HandlerFunc(func(w http.ResponseWriter,
		_ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}), protecao...)
	pedir := func(h http.Handler, alvo string) int {
		req := httptest.NewRequestWithContext(t.Context(),
			http.MethodGet, alvo, nil)
		req.Header.Set("Authorization", "Bearer t")
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		return rec.Code
	}
	fim := make(chan int)
	go func() { fim <- pedir(lenta, "/a") }()
	<-entrou
	if c := pedir(outra, "/b"); c != http.StatusServiceUnavailable {
		t.Fatalf("outra rota com a vaga ocupada: %d", c)
	}
	if c := pedir(outra, "/b?wait=1s"); c != http.StatusNoContent {
		t.Fatalf("long-poll barrado pelo limite: %d", c)
	}
	close(solta)
	<-fim
}
