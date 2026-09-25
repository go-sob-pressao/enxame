package postgres

import (
	"context"
	"time"

	"github.com/go-sob-pressao/enxame/internal/core/id"
	"github.com/go-sob-pressao/enxame/internal/core/job"
)

// livro:inicio fila

// Fila adapta o Store à fila que o pool de workers da Parte I espera.
// A interface daquela fila não recebe contexto — foi desenhada para a
// memória, onde nada espera —, então a Fila guarda o contexto do pool
// que a usa: vive exatamente o tempo dele (a exceção do Capítulo 5), e
// nunca deve ser reaproveitada por outro pool. Dívida declarada: a
// Parte IV passa o contexto por parâmetro.
type Fila struct {
	ctx   context.Context
	store *Store
	// Aviso de mudança: sem LISTEN/NOTIFY, o poll acorda a cada Aviso.
	Aviso time.Duration
}

// NewFila cria a fila para um pool que roda com ctx.
func NewFila(ctx context.Context, s *Store) *Fila {
	return &Fila{ctx: ctx, store: s, Aviso: 100 * time.Millisecond}
}

// livro:fim fila

// Fetch reserva o próximo job disponível.
func (f *Fila) Fetch(
	queue string,
	at time.Time,
	worker string,
) (job.Job, bool, error) {
	return f.store.Claim(f.ctx, queue, at, worker)
}

// Changed devolve um canal fechado depois de Aviso. O timer dispara uma
// função, sem goroutine esperando por ele.
func (f *Fila) Changed() <-chan struct{} {
	c := make(chan struct{})
	time.AfterFunc(f.Aviso, func() { close(c) })
	return c
}

// Complete registra o sucesso da tentativa.
func (f *Fila) Complete(jid id.JobID, at time.Time) error {
	_, err := f.store.Decide(f.ctx, jid,
		func(j job.Job) ([]job.Event, error) {
			return job.Complete(j, at)
		})
	return err
}

// Fail registra a falha da tentativa.
func (f *Fila) Fail(
	jid id.JobID,
	at time.Time,
	cause string,
	permanent bool,
	retryAt time.Time,
) error {
	_, err := f.store.Decide(f.ctx, jid,
		func(j job.Job) ([]job.Event, error) {
			return job.Fail(j, at, cause, permanent, retryAt)
		})
	return err
}

// Promote torna disponíveis os jobs cuja hora chegou.
func (f *Fila) Promote(at time.Time) (int, error) {
	return f.store.Promote(f.ctx, at)
}
