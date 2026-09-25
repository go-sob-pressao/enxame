// Package ingenuo — o fake de Store que aceitava tudo.
package ingenuo

import (
	"context"
	"sync"

	"github.com/go-sob-pressao/enxame/internal/core/id"
	"github.com/go-sob-pressao/enxame/internal/core/job"
	"github.com/go-sob-pressao/enxame/internal/store"
)

// livro:inicio fake-ingenuo

// Fake é o dublê que o time escreveu para os testes do serviço de
// pedidos: um mapa com trava. Faz tudo o que os testes pediam — e nada
// que ninguém pensou em pedir, como recusar uma unique_key repetida.
type Fake struct {
	mu   sync.Mutex
	jobs map[id.JobID]job.Job
}

// Insert grava o job.
func (f *Fake) Insert(
	_ context.Context,
	j job.Job,
	_ []job.Event,
) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.jobs == nil {
		f.jobs = map[id.JobID]job.Job{}
	}
	f.jobs[j.ID] = j
	return nil
}

// livro:fim fake-ingenuo

// Get devolve o job, sempre na versão 1.
func (f *Fake) Get(
	_ context.Context,
	jid id.JobID,
) (job.Job, int64, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	j, ok := f.jobs[jid]
	if !ok {
		return job.Job{}, 0, store.ErrNotFound
	}
	return j, 1, nil
}

// History não guarda nada: os testes do serviço nunca pediram.
func (f *Fake) History(context.Context, id.JobID) ([]job.Event, error) {
	return nil, nil
}

// Update sobrescreve, sem olhar a versão.
func (f *Fake) Update(
	_ context.Context,
	j job.Job,
	_ []job.Event,
	_ int64,
) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.jobs[j.ID] = j
	return nil
}

// Next devolve qualquer job disponível da fila.
func (f *Fake) Next(
	_ context.Context,
	queue string,
) (job.Job, int64, bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, j := range f.jobs {
		if j.Queue == queue && j.State == job.StateAvailable {
			return j, 1, true, nil
		}
	}
	return job.Job{}, 0, false, nil
}
