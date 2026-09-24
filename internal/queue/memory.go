package queue

import (
	"cmp"
	"errors"
	"slices"
	"time"

	"github.com/go-sob-pressao/enxame/internal/core/id"
	"github.com/go-sob-pressao/enxame/internal/core/job"
)

// ErrNotFound indica um job que a fila não conhece.
var ErrNotFound = errors.New("job não encontrado")

// livro:inicio memory

// Memory é a fila do M0: guarda projeção e histórico em memória, num
// único processo. NÃO é segura para uso concorrente — no M0, um job por
// vez. O Capítulo 3 vai chamá-la de várias goroutines e descobrir o
// preço.
type Memory struct {
	jobs    map[id.JobID]job.Job
	history map[id.JobID][]job.Event
}

// NewMemory cria uma fila vazia.
func NewMemory() *Memory {
	return &Memory{
		jobs:    map[id.JobID]job.Job{},
		history: map[id.JobID][]job.Event{},
	}
}

// livro:fim memory

func (m *Memory) gravar(j job.Job, evs []job.Event) (job.Job, error) {
	j, err := job.ApplyAll(j, evs)
	if err != nil {
		return j, err
	}
	m.jobs[j.ID] = j
	m.history[j.ID] = append(m.history[j.ID], evs...)
	return j, nil
}

func (m *Memory) buscar(jid id.JobID) (job.Job, error) {
	j, ok := m.jobs[jid]
	if !ok {
		return job.Job{}, ErrNotFound
	}
	return j, nil
}

// Insert enfileira um job.
func (m *Memory) Insert(s job.Spec, at time.Time) (job.Job, error) {
	if _, existe := m.jobs[s.ID]; existe {
		return job.Job{}, errors.New("job já existe: " + s.ID.String())
	}
	evs, j, err := job.Insert(s, at)
	if err != nil {
		return job.Job{}, err
	}
	return m.gravar(j, evs)
}

// livro:inicio fetch

// Fetch entrega o próximo job disponível da fila — menor prioridade
// numérica, depois o mais antigo — e o marca como em execução.
func (m *Memory) Fetch(
	queue string,
	at time.Time,
	worker string,
) (job.Job, bool, error) {
	var candidatos []job.Job
	for _, j := range m.jobs {
		if j.Queue == queue && j.State == job.StateAvailable {
			candidatos = append(candidatos, j)
		}
	}
	if len(candidatos) == 0 {
		return job.Job{}, false, nil
	}
	proximo := slices.MinFunc(candidatos, func(a, b job.Job) int {
		return cmp.Or(
			cmp.Compare(a.Priority, b.Priority),
			a.ScheduledAt.Compare(b.ScheduledAt),
			cmp.Compare(a.ID.String(), b.ID.String()),
		)
	})
	evs, err := job.Start(proximo, at, worker)
	if err != nil {
		return job.Job{}, false, err
	}
	j, err := m.gravar(proximo, evs)
	return j, err == nil, err
}

// livro:fim fetch

// Complete registra o sucesso da tentativa.
func (m *Memory) Complete(jid id.JobID, at time.Time) error {
	j, err := m.buscar(jid)
	if err != nil {
		return err
	}
	evs, err := job.Complete(j, at)
	if err != nil {
		return err
	}
	_, err = m.gravar(j, evs)
	return err
}

// Fail registra a falha da tentativa.
func (m *Memory) Fail(
	jid id.JobID,
	at time.Time,
	cause string,
	permanent bool,
	retryAt time.Time,
) error {
	j, err := m.buscar(jid)
	if err != nil {
		return err
	}
	evs, err := job.Fail(j, at, cause, permanent, retryAt)
	if err != nil {
		return err
	}
	_, err = m.gravar(j, evs)
	return err
}

// Promote torna disponíveis os jobs cujo horário chegou. No M0, quem
// chama é o laço do executor; o timer pump de verdade chega no Capítulo
// 11.
func (m *Memory) Promote(at time.Time) (int, error) {
	n := 0
	for _, j := range m.jobs {
		if (j.State == job.StateScheduled || j.State == job.StateRetryable) &&
			!at.Before(j.ScheduledAt) {
			evs, err := job.MakeAvailable(j, at)
			if err != nil {
				return n, err
			}
			if _, err := m.gravar(j, evs); err != nil {
				return n, err
			}
			n++
		}
	}
	return n, nil
}

// Get devolve a projeção e o histórico de um job.
func (m *Memory) Get(jid id.JobID) (job.Job, []job.Event, error) {
	j, err := m.buscar(jid)
	if err != nil {
		return job.Job{}, nil, err
	}
	return j, slices.Clone(m.history[jid]), nil
}
