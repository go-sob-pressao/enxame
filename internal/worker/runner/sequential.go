package runner

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/go-sob-pressao/enxame/internal/core/id"
	"github.com/go-sob-pressao/enxame/internal/core/job"
)

// Handler executa um job de um kind. Devolver nil conclui o job;
// devolver um erro agenda nova tentativa, exceto se o erro for
// Permanent.
type Handler func(ctx context.Context, j job.Job) error

// Queue é o que o executor precisa da fila. Declarada aqui, no
// consumidor (Cap. 12): o executor não conhece a implementação.
type Queue interface {
	Fetch(
		queue string,
		at time.Time,
		worker string,
	) (job.Job, bool, error)
	Complete(jid id.JobID, tentativa int, at time.Time) error
	Fail(
		jid id.JobID,
		tentativa int,
		at time.Time,
		cause string,
		permanent bool,
		retryAt time.Time,
	) error
	Promote(at time.Time) (int, error)
}

// ErrPermanent marca um erro que não adianta repetir.
var ErrPermanent = errors.New("erro permanente")

// Permanent embrulha err como permanente.
func Permanent(
	err error,
) error {
	return fmt.Errorf("%w: %w", ErrPermanent, err)
}

func isPermanent(err error) bool { return errors.Is(err, ErrPermanent) }

// livro:inicio sequential

// Sequential é o executor do M0: pega um job, executa, registra o
// resultado, e só então pega o próximo. Um job por vez, numa goroutine.
type Sequential struct {
	Queue      Queue
	QueueName  string
	Handlers   map[string]Handler
	Worker     string
	RetryDelay time.Duration
	Now        func() time.Time
}

// Drain executa até a fila não ter mais nada disponível.
func (s *Sequential) Drain(ctx context.Context) (int, error) {
	executados := 0
	for ctx.Err() == nil {
		if _, err := s.Queue.Promote(s.Now()); err != nil {
			return executados, err
		}
		j, ok, err := s.Queue.Fetch(s.QueueName, s.Now(), s.Worker)
		if err != nil || !ok {
			return executados, err
		}
		if err := s.executar(ctx, j); err != nil {
			return executados, err
		}
		executados++
	}
	return executados, ctx.Err()
}

func (s *Sequential) executar(ctx context.Context, j job.Job) error {
	return executar(ctx, s.Queue, s.Handlers, j, s.Now, s.RetryDelay)
}

// livro:fim sequential
