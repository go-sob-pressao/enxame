//go:build integration

package integration_test

import (
	"bytes"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	api "github.com/go-sob-pressao/enxame/internal/transport/http"
	"github.com/go-sob-pressao/enxame/test/testutil"
)

// livro:inicio bench-inserir

// BenchmarkInserirJob mede o caminho quente da API, POST /v1/jobs, do
// handler ao COMMIT no PostgreSQL, sem a rede entre cliente e servidor.
// ReportAllocs conta as alocações por requisição; b.Loop cuida do
// cronômetro e impede o compilador de sumir com o laço.
func BenchmarkInserirJob(b *testing.B) {
	db := testutil.Postgres(b)
	h := api.NovaAPI(db, map[string]string{"t": "loja"},
		slog.New(slog.DiscardHandler)).Handler()
	for _, n := range []int{0, 1_000, 100_000} {
		corpo := []byte(`{"queue":"q","kind":"k","args":{"dados":"` +
			strings.Repeat("x", n) + `"}}`)
		b.Run(fmt.Sprintf("args=%dB", n), func(b *testing.B) {
			b.ReportAllocs()
			b.SetBytes(int64(len(corpo)))
			for b.Loop() {
				r := httptest.NewRequestWithContext(b.Context(),
					http.MethodPost, "/v1/jobs", bytes.NewReader(corpo))
				r.Header.Set("Authorization", "Bearer t")
				w := httptest.NewRecorder()
				h.ServeHTTP(w, r)
				if w.Code != http.StatusCreated {
					b.Fatalf("%d: %s", w.Code, w.Body)
				}
			}
		})
	}
}

// livro:fim bench-inserir
