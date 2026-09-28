// Package p99 — Missão #6: o p99 que subiu oito vezes (Capítulo 29).
package p99_test

import (
	"bytes"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"syscall"
	"testing"
	"time"

	api "github.com/go-sob-pressao/enxame/internal/transport/http"
	"github.com/go-sob-pressao/enxame/test/testutil"
)

// kinds é a lista de kinds que a aplicação do chamado aceita: 400, de
// quatro sistemas.
func kinds() []string {
	var ks []string
	for _, s := range []string{"fatura", "email", "relatorio", "estoque"} {
		for i := range 100 {
			ks = append(ks, fmt.Sprintf("%s.tarefa-%03d", s, i))
		}
	}
	return ks
}

// livro:inicio missao-06-teste

// A API aceita 400 kinds. Cada POST /v1/jobs de um kind da lista tem de
// dar 201, e o de um kind de fora, 400. E o caminho quente tem de
// continuar barato: no máximo 200 alocações por requisição — o que ele
// fazia antes da lista.
func TestMissao(t *testing.T) {
	db := testutil.Postgres(t)
	a := api.NovaAPI(db, map[string]string{"t": "loja"},
		slog.New(slog.DiscardHandler))
	a.Kinds = kinds()
	h := a.Handler()
	if s := postar(t, h, "email.tarefa-042"); s != http.StatusCreated {
		t.Fatalf("kind da lista: %d", s)
	}
	if s := postar(t, h, "email.tarefa-999"); s != http.StatusBadRequest {
		t.Fatalf("kind de fora da lista: %d", s)
	}
	allocs := testing.AllocsPerRun(200, func() {
		postar(t, h, "estoque.tarefa-007")
	})
	semLista := api.NovaAPI(db, map[string]string{"t": "loja"},
		slog.New(slog.DiscardHandler)).Handler()
	cpu(t, semLista) // aquecimento
	t.Logf("%.0f alocações por requisição; CPU do processo por "+
		"requisição: %v com a lista, %v sem ela", allocs, cpu(t, h),
		cpu(t, semLista))
	if allocs > 200 {
		t.Fatalf("%.0f alocações por requisição; o limite é 200", allocs)
	}
}

// livro:fim missao-06-teste

func postar(t *testing.T, h http.Handler, kind string) int {
	corpo := []byte(`{"queue":"q","kind":"` + kind + `","args":{}}`)
	r := httptest.NewRequestWithContext(t.Context(), http.MethodPost,
		"/v1/jobs", bytes.NewReader(corpo))
	r.Header.Set("Authorization", "Bearer t")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	return w.Code
}

// cpu devolve o tempo de CPU do processo — usuário e sistema — por
// requisição, em 1.000 requisições seguidas. A latência, num notebook,
// é quase toda espera pelo banco; a CPU é o que o defeito consome.
func cpu(t *testing.T, h http.Handler) time.Duration {
	antes := usoDeCPU(t)
	for range 1000 {
		postar(t, h, "relatorio.tarefa-099")
	}
	return ((usoDeCPU(t) - antes) / 1000).Round(time.Microsecond)
}

func usoDeCPU(t *testing.T) time.Duration {
	var u syscall.Rusage
	if err := syscall.Getrusage(syscall.RUSAGE_SELF, &u); err != nil {
		t.Fatal(err)
	}
	return time.Duration(u.Utime.Nano() + u.Stime.Nano())
}
