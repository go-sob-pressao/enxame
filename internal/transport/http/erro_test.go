package http

import (
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-sob-pressao/enxame/internal/store"
)

func TestSemRecursosE503ComRetryAfter(t *testing.T) {
	a := &API{Log: slog.New(slog.DiscardHandler)}
	w := httptest.NewRecorder()
	r := httptest.NewRequestWithContext(t.Context(), http.MethodPost,
		"/v1/jobs", nil)
	a.erro(w, r, fmt.Errorf("inserir: %w", store.ErrSemRecursos))
	if w.Code != http.StatusServiceUnavailable ||
		w.Header().Get("Retry-After") != "5" {
		t.Fatalf("%d, Retry-After %q", w.Code,
			w.Header().Get("Retry-After"))
	}
}
