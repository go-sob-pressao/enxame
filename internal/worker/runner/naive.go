package runner

import (
	"context"
	"sync"
	"time"

	"github.com/go-sob-pressao/enxame/internal/core/job"
)

// livro:inicio naive

// Naive é o worker ingênuo do Capítulo 3: uma goroutine por job
// disponível, sem limite. Funciona no teste com dez jobs. Com cem mil,
// também "funciona" — até a memória do pod acabar.
type Naive struct {
	Queue      Queue
	QueueName  string
	Handlers   map[string]Handler
	Worker     string
	RetryDelay time.Duration
	Now        func() time.Time
}

// Drain dispara todos os jobs disponíveis ao mesmo tempo, espera todos
// terminarem e repete até não sobrar nada.
func (n *Naive) Drain(ctx context.Context) (int, error) {
	executados := 0
	for ctx.Err() == nil {
		if _, err := n.Queue.Promote(n.Now()); err != nil {
			return executados, err
		}
		var wg sync.WaitGroup
		disparados := 0
		for {
			j, ok, err := n.Queue.Fetch(n.QueueName, n.Now(), n.Worker)
			if err != nil {
				return executados, err
			}
			if !ok {
				break
			}
			wg.Go(func() {
				_ = executar(
					ctx,
					n.Queue,
					n.Handlers,
					j,
					n.Now,
					n.RetryDelay,
				)
			})
			disparados++
		}
		wg.Wait()
		if disparados == 0 {
			return executados, nil
		}
		executados += disparados
	}
	return executados, ctx.Err()
}

// livro:fim naive

// executar roda o handler do job e registra o resultado na fila.
func executar(
	ctx context.Context,
	q Queue,
	hs map[string]Handler,
	j job.Job,
	agora func() time.Time,
	retry time.Duration,
) error {
	h, ok := hs[j.Kind]
	if !ok {
		return q.Fail(
			j.ID,
			j.Attempt,
			agora(),
			"kind sem handler: "+j.Kind,
			true,
			time.Time{},
		)
	}
	if err := h(ctx, j); err != nil {
		t := agora()
		return q.Fail(
			j.ID,
			j.Attempt,
			t,
			err.Error(),
			isPermanent(err),
			t.Add(retry),
		)
	}
	return q.Complete(j.ID, j.Attempt, agora())
}
