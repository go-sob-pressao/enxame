//go:build integration

package integration_test

import (
	"errors"
	"testing"
	"time"

	"github.com/go-sob-pressao/enxame/internal/core/job"
	"github.com/go-sob-pressao/enxame/internal/store/postgres"
	"github.com/go-sob-pressao/enxame/test/testutil"
)

// livro:inicio tentativa-zumbi

// A tentativa 1 é resgatada viva — o worker estava pausado — e o job
// volta a rodar, na tentativa 2, em outro worker. Quando o primeiro
// acorda e tenta concluir, é recusado; o histórico só registra o fim da
// tentativa corrente.
func TestTentativaResgatadaNaoConclui(t *testing.T) {
	s := postgres.New(testutil.Postgres(t))
	jid := enfileirar(t, s, 1)[0]
	t0 := time.Now()
	velho := postgres.NewFila(t.Context(), s)
	j1, _, err := velho.Fetch("q", t0, "w-pausado")
	if err != nil || j1.Attempt != 1 {
		t.Fatalf("primeira reserva: %v %v", j1.Attempt, err)
	}
	// O resgate acha a tentativa sem sinal de vida e a devolve à fila.
	if n, err := s.Rescue(t.Context(), t0.Add(time.Minute),
		t0.Add(time.Second)); n != 1 || err != nil {
		t.Fatalf("resgate: %d %v", n, err)
	}
	_, err = s.Promote(t.Context(), t0.Add(2*time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	novo := postgres.NewFila(t.Context(), s)
	j2, _, err := novo.Fetch("q", t0.Add(2*time.Minute), "w-novo")
	if err != nil || j2.Attempt != 2 {
		t.Fatalf("segunda reserva: %v %v", j2.Attempt, err)
	}
	err = velho.Complete(jid, j1.Attempt, t0.Add(3*time.Minute))
	if !errors.Is(err, job.ErrInvalidTransition) {
		t.Fatalf("o zumbi concluiu o job: %v", err)
	}
	t.Logf("fim da tentativa 1 recusado: %v", err)
	if err := novo.Complete(jid, j2.Attempt,
		t0.Add(3*time.Minute)); err != nil {
		t.Fatal(err)
	}
}

// livro:fim tentativa-zumbi
