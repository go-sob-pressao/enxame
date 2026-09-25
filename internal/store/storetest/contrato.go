// Package storetest é a suíte de contrato do armazenamento de jobs:
// uma única bateria de testes que toda implementação — memória, SQLite,
// Postgres — precisa aprovar.
package storetest

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/go-sob-pressao/enxame/internal/core/id"
	"github.com/go-sob-pressao/enxame/internal/core/job"
	"github.com/go-sob-pressao/enxame/internal/store"
	"github.com/go-sob-pressao/enxame/internal/store/serde"
)

// Store é o contrato que a suíte verifica. É o mesmo método a método
// que o motor declara em internal/engine: a suíte é mais um consumidor.
type Store interface {
	Insert(ctx context.Context, j job.Job, evs []job.Event) error
	Get(ctx context.Context, jid id.JobID) (job.Job, int64, error)
	History(ctx context.Context, jid id.JobID) ([]job.Event, error)
	Update(
		ctx context.Context,
		j job.Job,
		evs []job.Event,
		version int64,
	) error
	Next(
		ctx context.Context,
		queue string,
	) (job.Job, int64, bool, error)
}

// Nova cria uma implementação vazia, isolada, para um subteste.
type Nova func(t *testing.T) Store

// t0 tem precisão de segundos: todas as implementações a guardam sem
// perda (o Postgres guarda microssegundos).
var t0 = time.Date(2026, 9, 25, 12, 0, 0, 0, time.UTC)

// livro:inicio contrato

// Run executa o contrato inteiro contra a implementação que nova cria.
func Run(t *testing.T, nova Nova) {
	casos := []struct {
		nome  string
		teste func(t *testing.T, s Store)
	}{
		{"grava e lê o job e o histórico", gravaELe},
		{"job inexistente é ErrNotFound", inexistente},
		{"unique_key repetida é recusada", chaveRepetida},
		{"unique_key volta a valer após cancelamento", chaveLiberada},
		{"unique_key não volta após conclusão", chaveConcluida},
		{"versão desatualizada é recusada", versaoVelha},
		{"Next respeita prioridade e horário", ordemDoNext},
		{"Next ignora o que não está disponível", nextSoDisponivel},
	}
	for _, c := range casos {
		t.Run(c.nome, func(t *testing.T) { c.teste(t, nova(t)) })
	}
}

// livro:fim contrato

// novo insere um job criado pelo domínio e o devolve.
func novo(t *testing.T, s Store, n int, spec job.Spec) job.Job {
	t.Helper()
	spec.ID = idDe(n)
	if spec.Queue == "" {
		spec.Queue = "q"
	}
	if spec.Kind == "" {
		spec.Kind = "eco"
	}
	if spec.Namespace == "" {
		spec.Namespace = "ns"
	}
	evs, j, err := job.Insert(spec, t0)
	if err != nil {
		t.Fatal(err)
	}
	if j, err = job.ApplyAll(j, evs); err != nil {
		t.Fatal(err)
	}
	if err := s.Insert(t.Context(), j, evs); err != nil {
		t.Fatalf("Insert: %v", err)
	}
	return j
}

// avancar aplica uma decisão do domínio e grava com a versão lida.
func avancar(
	t *testing.T,
	s Store,
	jid id.JobID,
	decide func(job.Job) ([]job.Event, error),
) job.Job {
	t.Helper()
	j, v, err := s.Get(t.Context(), jid)
	if err != nil {
		t.Fatal(err)
	}
	evs, err := decide(j)
	if err != nil {
		t.Fatal(err)
	}
	if j, err = job.ApplyAll(j, evs); err != nil {
		t.Fatal(err)
	}
	if err := s.Update(t.Context(), j, evs, v); err != nil {
		t.Fatalf("Update: %v", err)
	}
	return j
}

func idDe(n int) id.JobID {
	jid, err := id.ParseJobID(
		fmt.Sprintf("0192a3b4-0000-7000-8000-%012d", n),
	)
	if err != nil {
		panic(err)
	}
	return jid
}

func gravaELe(t *testing.T, s Store) {
	j := novo(t, s, 1, job.Spec{
		Args: []byte(
			`{"pedido": 9007199254740993, "itens": ["a", "b"]}`,
		),
		UniqueKey: "pedido-42",
		Priority:  1,
	})
	got, v, err := s.Get(t.Context(), j.ID)
	if err != nil {
		t.Fatal(err)
	}
	if v <= 0 {
		t.Errorf("versão %d depois do Insert", v)
	}
	mesmoJob(t, got, j)
	hist, err := s.History(t.Context(), j.ID)
	if err != nil || len(hist) != 1 ||
		hist[0].Type != job.EventInserted || !hist[0].At.Equal(t0) {
		t.Fatalf("histórico %+v, err %v", hist, err)
	}
}

func inexistente(t *testing.T, s Store) {
	if _, _, err := s.Get(t.Context(), idDe(99)); !errors.Is(
		err,
		store.ErrNotFound,
	) {
		t.Fatalf("Get de inexistente: %v", err)
	}
}

// livro:inicio contrato-unicidade

func chaveRepetida(t *testing.T, s Store) {
	novo(t, s, 1, job.Spec{UniqueKey: "pedido-42"})
	spec := job.Spec{
		ID: idDe(2), Namespace: "ns", Queue: "q", Kind: "eco",
		UniqueKey: "pedido-42",
	}
	evs, j, _ := job.Insert(spec, t0)
	j, _ = job.ApplyAll(j, evs)
	if err := s.Insert(t.Context(), j, evs); !errors.Is(
		err,
		store.ErrDuplicate,
	) {
		t.Fatalf("segundo job com a mesma chave: %v", err)
	}
	outro := j
	outro.Namespace = "outro-ns" // a chave é única por namespace
	if err := s.Insert(t.Context(), outro, evs); err != nil {
		t.Fatalf("mesma chave em outro namespace: %v", err)
	}
}

func chaveLiberada(t *testing.T, s Store) {
	j := novo(t, s, 1, job.Spec{UniqueKey: "pedido-42"})
	avancar(t, s, j.ID, func(j job.Job) ([]job.Event, error) {
		return job.Cancel(j, t0)
	})
	novo(t, s, 2, job.Spec{UniqueKey: "pedido-42"}) // não pode falhar
}

func chaveConcluida(t *testing.T, s Store) {
	j := novo(t, s, 1, job.Spec{UniqueKey: "pedido-42"})
	avancar(t, s, j.ID, func(j job.Job) ([]job.Event, error) {
		return job.Start(j, t0, "w1")
	})
	avancar(t, s, j.ID, func(j job.Job) ([]job.Event, error) {
		return job.Complete(j, t0)
	})
	spec := job.Spec{
		ID: idDe(2), Namespace: "ns", Queue: "q", Kind: "eco",
		UniqueKey: "pedido-42",
	}
	evs, j2, _ := job.Insert(spec, t0)
	j2, _ = job.ApplyAll(j2, evs)
	if err := s.Insert(t.Context(), j2, evs); !errors.Is(
		err,
		store.ErrDuplicate,
	) {
		t.Fatalf("chave de job concluído foi reaproveitada: %v", err)
	}
}

// livro:fim contrato-unicidade

// livro:inicio contrato-versao

// Dois workers leem a mesma versão; o segundo a gravar perde. É o
// fencing no nível do job: quem trabalha com uma versão velha não
// sobrescreve o que o outro gravou.
func versaoVelha(t *testing.T, s Store) {
	j := novo(t, s, 1, job.Spec{})
	lido, v, err := s.Get(t.Context(), j.ID)
	if err != nil {
		t.Fatal(err)
	}
	avancar(t, s, j.ID, func(j job.Job) ([]job.Event, error) {
		return job.Start(j, t0, "w1")
	})
	evs, _ := job.Start(lido, t0, "w2")
	atrasado, _ := job.ApplyAll(lido, evs)
	if err := s.Update(t.Context(), atrasado, evs, v); !errors.Is(
		err,
		store.ErrConflict,
	) {
		t.Fatalf("gravação com versão velha: %v", err)
	}
	if got, _, _ := s.Get(t.Context(), j.ID); got.AttemptedBy != "w1" {
		t.Fatalf("o job ficou com %q", got.AttemptedBy)
	}
}

// livro:fim contrato-versao

func ordemDoNext(t *testing.T, s Store) {
	novo(t, s, 1, job.Spec{Priority: 3})
	urgente := novo(t, s, 2, job.Spec{Priority: 1})
	novo(t, s, 3, job.Spec{Priority: 1, RunAt: t0.Add(time.Hour)})
	got, _, ok, err := s.Next(t.Context(), "q")
	if err != nil || !ok || got.ID != urgente.ID {
		t.Fatalf("Next = %v %v %v; esperado o job 2", got.ID, ok, err)
	}
}

func nextSoDisponivel(t *testing.T, s Store) {
	j := novo(t, s, 1, job.Spec{})
	novo(t, s, 2, job.Spec{Queue: "outra"})
	avancar(t, s, j.ID, func(j job.Job) ([]job.Event, error) {
		return job.Start(j, t0, "w1")
	})
	if _, _, ok, err := s.Next(t.Context(), "q"); ok || err != nil {
		t.Fatalf(
			"Next com o único job da fila em execução: %v %v",
			ok,
			err,
		)
	}
}

// mesmoJob compara as projeções; Args como JSON equivalente, porque o
// Postgres guarda JSONB e devolve o JSON normalizado.
func mesmoJob(t *testing.T, got, want job.Job) {
	t.Helper()
	if !jsonIgual(t, got.Args, want.Args) {
		t.Errorf("Args %s, esperado %s", got.Args, want.Args)
	}
	got.Args, want.Args = nil, nil
	if !got.ScheduledAt.Equal(want.ScheduledAt) {
		t.Errorf(
			"ScheduledAt %v, esperado %v",
			got.ScheduledAt,
			want.ScheduledAt,
		)
	}
	got.ScheduledAt, want.ScheduledAt = time.Time{}, time.Time{}
	if fmt.Sprintf("%+v", got) != fmt.Sprintf("%+v", want) {
		t.Errorf("job\n got %+v\nwant %+v", got, want)
	}
}

// jsonIgual compara número a número, pela forma canônica do serde.
// (Até o Capítulo 13, esta função decodificava para any e comparava os
// float64: dois ids de 19 dígitos diferentes passavam por iguais.)
func jsonIgual(t *testing.T, a, b []byte) bool {
	t.Helper()
	igual, err := serde.Equal(a, b)
	if err != nil {
		t.Fatalf("Args inválido: %v", err)
	}
	return igual
}
