package job_test

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/go-sob-pressao/enxame/internal/core/job"
)

// acao é uma decisão do domínio aplicada a um job, do jeito que a
// especificação a nomeia.
type acao struct {
	nome   string
	decide func(j job.Job) ([]job.Event, error)
}

var (
	iniciar = acao{"iniciar", func(j job.Job) ([]job.Event, error) {
		return job.Start(j, j.ScheduledAt, "w1")
	}}
	concluir = acao{"concluir", func(j job.Job) ([]job.Event, error) {
		return job.Complete(j, t0)
	}}
	falhar = acao{"falhar", func(j job.Job) ([]job.Event, error) {
		return job.Fail(j, t0, "erro", false, t0.Add(time.Minute))
	}}
	falharDeVez = acao{"falhar de vez", func(j job.Job) ([]job.Event, error) {
		return job.Fail(j, t0, "erro", true, time.Time{})
	}}
	liberar = acao{"liberar", func(j job.Job) ([]job.Event, error) {
		return job.MakeAvailable(j, j.ScheduledAt)
	}}
	cancelar = acao{"cancelar", func(j job.Job) ([]job.Event, error) {
		return job.Cancel(j, t0)
	}}
	resgatar = acao{"resgatar", func(j job.Job) ([]job.Event, error) {
		return job.Rescue(j, t0)
	}}
)

var acoes = []acao{
	iniciar, concluir, falhar, falharDeVez, liberar, cancelar, resgatar,
}

var estados = []job.State{
	job.StateScheduled, job.StateAvailable, job.StateRunning,
	job.StateRetryable, job.StateCompleted, job.StateDiscarded,
	job.StateCancelled,
}

// livro:inicio especificacao

// especificacao é a máquina de estados do job, escrita como tabela: as
// únicas transições válidas. Qualquer par (estado, ação) que não está
// aqui precisa ser recusado com ErrInvalidTransition.
var especificacao = []struct {
	de   job.State
	acao acao
	para job.State
}{
	{job.StateScheduled, liberar, job.StateAvailable},
	{job.StateScheduled, cancelar, job.StateCancelled},
	{job.StateAvailable, iniciar, job.StateRunning},
	{job.StateAvailable, cancelar, job.StateCancelled},
	{job.StateRunning, concluir, job.StateCompleted},
	{job.StateRunning, falhar, job.StateRetryable},
	{job.StateRunning, falharDeVez, job.StateDiscarded},
	{job.StateRunning, resgatar, job.StateAvailable},
	{job.StateRetryable, liberar, job.StateAvailable},
	{job.StateRetryable, cancelar, job.StateCancelled},
}

// TestMaquinaDeEstados percorre todos os pares (estado, ação): os da
// especificação levam ao estado previsto; os outros são recusados.
func TestMaquinaDeEstados(t *testing.T) {
	for _, de := range estados {
		for _, a := range acoes {
			nome := fmt.Sprintf("%s/%s", de, a.nome)
			t.Run(nome, func(t *testing.T) {
				j := emEstado(t, de)
				evs, err := a.decide(j)
				para, valida := destino(de, a)
				if !valida {
					if !errors.Is(err, job.ErrInvalidTransition) {
						t.Fatalf("deveria ser recusada; err=%v", err)
					}
					return
				}
				j = aplicar(t, j, evs, err)
				if j.State != para {
					t.Fatalf("foi para %q, esperado %q", j.State, para)
				}
			})
		}
	}
}

// livro:fim especificacao

func destino(de job.State, a acao) (job.State, bool) {
	for _, e := range especificacao {
		if e.de == de && e.acao.nome == a.nome {
			return e.para, true
		}
	}
	return "", false
}

// livro:inicio fixture

// emEstado devolve um job no estado pedido, construído pelas próprias
// transições do domínio: nenhum campo é preenchido à mão, então a
// fixture nunca descreve um job que o domínio não produziria.
func emEstado(t *testing.T, s job.State) job.Job {
	t.Helper()
	passo := func(j job.Job, a acao) job.Job {
		t.Helper()
		evs, err := a.decide(j)
		return aplicar(t, j, evs, err)
	}
	switch s {
	case job.StateScheduled:
		return novo(t, job.Spec{RunAt: t0.Add(time.Hour)})
	case job.StateAvailable:
		return novo(t, job.Spec{})
	case job.StateRunning:
		return passo(emEstado(t, job.StateAvailable), iniciar)
	case job.StateRetryable:
		return passo(emEstado(t, job.StateRunning), falhar)
	case job.StateCompleted:
		return passo(emEstado(t, job.StateRunning), concluir)
	case job.StateDiscarded:
		return passo(emEstado(t, job.StateRunning), falharDeVez)
	case job.StateCancelled:
		return passo(emEstado(t, job.StateAvailable), cancelar)
	}
	t.Fatalf("estado desconhecido: %q", s)
	return job.Job{}
}

// livro:fim fixture

// livro:inicio especificacao-insert

// As regras do pedido de enfileiramento, uma linha cada: cada caso
// parte de um pedido válido e muda uma coisa só.
func TestInsertValidaOPedido(t *testing.T) {
	casos := []struct {
		nome   string
		mudar  func(*job.Spec)
		aceita bool
	}{
		{"pedido mínimo", func(*job.Spec) {}, true},
		{"sem fila", func(s *job.Spec) { s.Queue = "" }, false},
		{"sem kind", func(s *job.Spec) { s.Kind = "" }, false},
		{"prioridade 5", func(s *job.Spec) { s.Priority = 5 }, false},
		{
			"tentativas negativas",
			func(s *job.Spec) { s.MaxAttempts = -1 },
			false,
		},
	}
	for _, c := range casos {
		t.Run(c.nome, func(t *testing.T) {
			s := job.Spec{ID: idFixo, Queue: "padrao", Kind: "eco"}
			c.mudar(&s)
			_, _, err := job.Insert(s, t0)
			if aceitou := err == nil; aceitou != c.aceita {
				t.Fatalf(
					"aceitou=%v, esperado %v (err=%v)",
					aceitou, c.aceita, err,
				)
			}
		})
	}
}

// livro:fim especificacao-insert
