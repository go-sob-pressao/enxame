package http_test

import (
	"bytes"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	api "github.com/go-sob-pressao/enxame/internal/transport/http"
)

// O log de cada requisição traz o padrão da rota, que só o ServeMux
// conhece — lá dentro, depois do middleware que registra.
func TestRegistrarAnotaARota(t *testing.T) {
	var log bytes.Buffer
	a := api.NovaAPI(nil, nil, slog.New(slog.NewJSONHandler(&log, nil)))
	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet,
		"/healthz", nil)
	a.Handler().ServeHTTP(httptest.NewRecorder(), req)
	if !strings.Contains(log.String(), `"rota":"GET /healthz"`) {
		t.Fatalf("log sem a rota: %s", log.String())
	}
}
