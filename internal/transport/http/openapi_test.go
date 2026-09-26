package http_test

import (
	"bytes"
	"log/slog"
	"os"
	"testing"

	api "github.com/go-sob-pressao/enxame/internal/transport/http"
)

// O arquivo do repositório é o que o código gera: uma rota nova sem
// make openapi reprova aqui.
func TestOpenAPIEmDia(t *testing.T) {
	a := api.NovaAPI(nil, nil, slog.New(slog.DiscardHandler))
	gerado, err := a.OpenAPI("v1")
	if err != nil {
		t.Fatal(err)
	}
	gravado, err := os.ReadFile("../../../api/openapi.json")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(bytes.TrimSpace(gravado), bytes.TrimSpace(gerado)) {
		t.Fatal("api/openapi.json desatualizado: rode make openapi")
	}
}
