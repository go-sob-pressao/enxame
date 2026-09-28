package observ_test

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/go-sob-pressao/enxame/internal/observ"
)

func TestVooGravaARequisicaoLenta(t *testing.T) {
	v := &observ.Voo{Limiar: 50 * time.Millisecond, Dir: t.TempDir(),
		Intervalo: time.Minute, Log: slog.New(slog.DiscardHandler)}
	if err := v.Ligar(); err != nil {
		t.Fatal(err)
	}
	defer v.Desligar()
	h := v.Middleware(http.HandlerFunc(func(http.ResponseWriter,
		*http.Request) {
		<-time.After(100 * time.Millisecond)
	}))
	for range 2 { // a segunda cai dentro do intervalo: não grava
		r := httptest.NewRequestWithContext(t.Context(), http.MethodGet,
			"/", nil)
		h.ServeHTTP(httptest.NewRecorder(), r)
	}
	arqs, _ := filepath.Glob(filepath.Join(v.Dir, "voo-*.trace"))
	if len(arqs) != 1 {
		t.Fatalf("%d traces gravados; esperava 1", len(arqs))
	}
	if info, err := os.Stat(arqs[0]); err != nil || info.Size() == 0 {
		t.Fatalf("trace vazio: %v", err)
	}
}
