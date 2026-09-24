//go:build defeito

package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"runtime"
	"sync"
	"testing"
)

// Cada requisição que estoura o prazo deixa uma goroutine bloqueada
// para sempre no envio: ninguém mais lê o canal. As 50 requisições
// correm em paralelo, então o teste espera os 500 ms uma vez só.
func TestCadaTimeoutDeixaUmaGoroutine(t *testing.T) {
	lento := make(chan struct{})
	consultar := func(context.Context, string) (int, error) { <-lento; return 1, nil }
	h := handlerFrete(consultar)

	antes := runtime.NumGoroutine()
	const n = 50
	var wg sync.WaitGroup
	for range n {
		wg.Go(func() {
			rec := httptest.NewRecorder()
			req := httptest.NewRequestWithContext(
				t.Context(),
				http.MethodGet,
				"/frete/01001000",
				nil,
			)
			req.SetPathValue("cep", "01001000")
			h.ServeHTTP(rec, req)
			if rec.Code != http.StatusGatewayTimeout {
				t.Errorf("status %d, esperado 504", rec.Code)
			}
		})
	}
	wg.Wait()
	close(lento) // o frete finalmente responde… para ninguém

	depois := runtime.NumGoroutine()
	for range 1000 {
		runtime.Gosched()
		depois = runtime.NumGoroutine()
	}
	if depois-antes < n {
		t.Fatalf(
			"goroutines: antes %d, depois %d; o enigma prevê %d a mais",
			antes,
			depois,
			n,
		)
	}
	t.Logf(
		"%d requisições com timeout, %d goroutines presas no envio",
		n,
		depois-antes,
	)
}
