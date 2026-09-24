//go:build !defeito

package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"go.uber.org/goleak"
)

func TestTimeoutNaoDeixaGoroutine(t *testing.T) {
	defer goleak.VerifyNone(t)
	consultar := func(ctx context.Context, _ string) (int, error) { <-ctx.Done(); return 0, ctx.Err() }
	h := handlerFrete(consultar)
	var wg sync.WaitGroup
	for range 50 {
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
				t.Errorf("status %d", rec.Code)
			}
		})
	}
	wg.Wait()
}
