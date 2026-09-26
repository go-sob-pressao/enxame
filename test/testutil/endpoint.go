package testutil

import (
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/go-sob-pressao/enxame/pkg/webhook"
)

// Endpoint é um receptor de webhooks para testes, que verifica cada
// assinatura e responde o que o teste mandar.
type Endpoint struct {
	URL string

	mu        sync.Mutex
	responder func(n int) int // status da n-ésima requisição (1, 2…)
	recebidas int
	validas   int
	ids       map[string]int // webhook-id → vezes
}

// NovoEndpoint sobe o receptor. responder decide o status de cada
// requisição pelo número dela; um 429 vai com Retry-After: 2.
func NovoEndpoint(
	t testing.TB,
	segredo string,
	responder func(n int) int,
) *Endpoint {
	t.Helper()
	e := &Endpoint{responder: responder, ids: map[string]int{}}
	srv := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			corpo, _ := io.ReadAll(r.Body)
			id, err := webhook.Verificar(segredo, r.Header, corpo,
				time.Now())
			e.mu.Lock()
			e.recebidas++
			n := e.recebidas
			if err == nil {
				e.validas++
				e.ids[id]++
			}
			e.mu.Unlock()
			if err != nil {
				http.Error(w, err.Error(), http.StatusUnauthorized)
				return
			}
			status := e.responder(n)
			if status == http.StatusTooManyRequests {
				w.Header().Set("Retry-After", "2") // segundos
			}
			w.WriteHeader(status)
		}))
	t.Cleanup(srv.Close)
	e.URL = srv.URL
	return e
}

// Contagem devolve quantas requisições chegaram, quantas com
// assinatura válida, e quantos webhook-id distintos.
func (e *Endpoint) Contagem() (recebidas, validas, ids int) {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.recebidas, e.validas, len(e.ids)
}
