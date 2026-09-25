package job_test

import (
	"errors"
	"testing"
	"time"

	"pgregory.net/rapid"

	"github.com/go-sob-pressao/enxame/internal/core/job"
)

// porNome permite ao rapid sortear ações pelo nome, que ele sabe
// imprimir quando encontra um contraexemplo.
var porNome = func() map[string]acao {
	m := map[string]acao{}
	for _, a := range acoes {
		m[a.nome] = a
	}
	return m
}()

var nomes = func() []string {
	var n []string
	for _, a := range acoes {
		n = append(n, a.nome)
	}
	return n
}()

// livro:inicio propriedade

// TestPropriedadesDaMaquina sorteia um job e uma sequência de até 50
// ações, aplica as que o domínio aceita e confere os invariantes depois
// de cada passo. Quando falha, o rapid encolhe a sequência até a menor
// que ainda falha.
func TestPropriedadesDaMaquina(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		spec := job.Spec{
			ID: idFixo, Queue: "q", Kind: "eco",
			MaxAttempts: rapid.IntRange(1, 4).Draw(t, "max"),
		}
		if rapid.Bool().Draw(t, "agendado") {
			spec.RunAt = t0.Add(time.Hour)
		}
		evs, base, err := job.Insert(spec, t0)
		if err != nil {
			t.Fatal(err)
		}
		j, _ := job.ApplyAll(base, evs)
		historico := evs
		passos := rapid.SliceOfN(rapid.SampledFrom(nomes), 0, 50).
			Draw(t, "acoes")
		for _, nome := range passos {
			antes := j
			novos, err := porNome[nome].decide(j)
			if err != nil {
				if !errors.Is(err, job.ErrInvalidTransition) {
					t.Fatalf("%s: erro inesperado %v", nome, err)
				}
				continue
			}
			if j, err = job.ApplyAll(j, novos); err != nil {
				t.Fatal(err)
			}
			historico = append(historico, novos...)
			invariantes(t, nome, antes, j, base, historico)
		}
	})
}

// livro:fim propriedade

// livro:inicio invariantes

func invariantes(
	t *rapid.T,
	nome string,
	antes, j, base job.Job,
	historico []job.Event,
) {
	if antes.State.Final() {
		t.Fatalf("%s mudou um job já final (%s)", nome, antes.State)
	}
	if j.Attempt < antes.Attempt || j.Attempt > j.MaxAttempts {
		t.Fatalf("%s: tentativa %d (antes %d, máx %d)",
			nome, j.Attempt, antes.Attempt, j.MaxAttempts)
	}
	if j.State.Final() == j.FinalizedAt.IsZero() {
		t.Fatalf("%s: estado %s com FinalizedAt %v",
			nome, j.State, j.FinalizedAt)
	}
	if j.State == job.StateRunning && j.AttemptedBy == "" {
		t.Fatalf("%s: em execução sem worker", nome)
	}
	refeito, err := job.ApplyAll(base, historico)
	if err != nil || refeito.State != j.State ||
		refeito.Attempt != j.Attempt ||
		refeito.LastError != j.LastError {
		t.Fatalf("%s: o histórico não reproduz a projeção", nome)
	}
}

// livro:fim invariantes
