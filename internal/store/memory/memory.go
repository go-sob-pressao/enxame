package memory

import (
	"cmp"
	"context"
	"fmt"
	"slices"
	"sync"

	"github.com/go-sob-pressao/enxame/internal/core/id"
	"github.com/go-sob-pressao/enxame/internal/core/job"
	"github.com/go-sob-pressao/enxame/internal/store"
)

// Store guarda jobs em memória, com a mesma semântica do Postgres:
// unicidade parcial de unique_key, lock otimista por versão e a mesma
// ordem de busca. A suíte de contrato garante que não divirjam.
type Store struct {
	mu   sync.Mutex
	jobs map[id.JobID]*registro
}

type registro struct {
	job       job.Job
	versao    int64
	historico []job.Event
}

// New cria um store vazio.
func New() *Store { return &Store{jobs: map[id.JobID]*registro{}} }

// Insert grava um job novo e o primeiro histórico.
func (s *Store) Insert(
	_ context.Context,
	j job.Job,
	evs []job.Event,
) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, existe := s.jobs[j.ID]; existe {
		return fmt.Errorf("%w: job_id %s", store.ErrDuplicate, j.ID)
	}
	if err := s.verificarUnicidade(j); err != nil {
		return err
	}
	s.jobs[j.ID] = &registro{
		job:       j,
		versao:    1,
		historico: slices.Clone(evs),
	}
	return nil
}

// Get devolve a projeção e a versão.
func (s *Store) Get(
	_ context.Context,
	jid id.JobID,
) (job.Job, int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	r, ok := s.jobs[jid]
	if !ok {
		return job.Job{}, 0, store.ErrNotFound
	}
	return r.job, r.versao, nil
}

// History devolve os eventos do job, em ordem.
func (s *Store) History(
	_ context.Context,
	jid id.JobID,
) ([]job.Event, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	r, ok := s.jobs[jid]
	if !ok {
		return nil, store.ErrNotFound
	}
	return slices.Clone(r.historico), nil
}

// Update grava a projeção nova se a versão ainda for version.
func (s *Store) Update(
	_ context.Context,
	j job.Job,
	evs []job.Event,
	version int64,
) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	r, ok := s.jobs[j.ID]
	if !ok {
		return store.ErrNotFound
	}
	if r.versao != version {
		return fmt.Errorf(
			"%w: gravada %d, esperada %d",
			store.ErrConflict,
			r.versao,
			version,
		)
	}
	if err := s.verificarUnicidade(j); err != nil {
		return err
	}
	r.job = j
	r.versao++
	r.historico = append(r.historico, evs...)
	return nil
}

// Next devolve o job disponível de menor prioridade numérica, depois o
// mais antigo, depois o de menor id — a mesma ordem do índice do
// Postgres.
func (s *Store) Next(
	_ context.Context,
	queue string,
) (job.Job, int64, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var melhor *registro
	for _, r := range s.jobs {
		j := r.job
		if j.Queue != queue || j.State != job.StateAvailable {
			continue
		}
		if melhor == nil || antes(j, melhor.job) {
			melhor = r
		}
	}
	if melhor == nil {
		return job.Job{}, 0, false, nil
	}
	return melhor.job, melhor.versao, true, nil
}

func antes(a, b job.Job) bool {
	return cmp.Or(
		cmp.Compare(a.Priority, b.Priority),
		a.ScheduledAt.Compare(b.ScheduledAt),
		cmp.Compare(a.ID.String(), b.ID.String()),
	) < 0
}
