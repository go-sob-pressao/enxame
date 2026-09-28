package enxame_test

import (
	"errors"
	"testing"
	"time"

	"github.com/go-sob-pressao/enxame/pkg/enxame"
)

// livro:inicio config-recusada

// A configuração da Missão #7 — resgate de 2 s, relatórios de 3 s —
// não sobe mais: o prazo de cada tentativa e o do resgate são dois
// números, e o resgate precisa vir depois.
func TestResgateAntesDoPrazoERecusado(t *testing.T) {
	w := enxame.New(nil, "loja").NewWorker(enxame.WorkerConfig{
		RescueAfter: 2 * time.Second})
	err := w.Run(t.Context())
	if !errors.Is(err, enxame.ErrConfig) {
		t.Fatalf("Run = %v; queria ErrConfig", err)
	}
	t.Log(err)
}

// livro:fim config-recusada
