package queue

import (
	"context"
	"time"

	"github.com/go-sob-pressao/enxame/internal/core/job"
)

// Notifier avisa que a fila mudou. Changed devolve um canal que é
// fechado na próxima mudança: fechar é o único jeito de acordar TODOS
// os que esperam num canal, porque um envio acorda um só.
type Notifier interface {
	Changed() <-chan struct{}
}

// PollConfig configura o long-poll.
type PollConfig struct {
	Queue  string
	Worker string
	// quanto esperar por trabalho antes de devolver vazio
	PollTimeout time.Duration
	Now         func() time.Time
}

// livro:inicio longpoll

// Poll espera até existir um job, o prazo do poll vencer ou o contexto
// ser cancelado — o que vier primeiro. As três vias são indispensáveis:
//
//   - sem ctx.Done(), o desligamento espera o prazo inteiro de cada
//     worker;
//   - sem o prazo, o worker nunca volta para renovar heartbeat ou ver
//     config;
//   - sem a notificação, todo job novo espera o prazo inteiro para
//     começar.
func Poll(
	ctx context.Context,
	src Source,
	n Notifier,
	cfg PollConfig,
) (job.Job, bool, error) {
	prazo := time.NewTimer(cfg.PollTimeout)
	defer prazo.Stop()
	for {
		// antes do Fetch: nenhuma mudança entre os dois se perde
		mudou := n.Changed()
		j, ok, err := src.Fetch(cfg.Queue, cfg.Now(), cfg.Worker)
		if err != nil || ok {
			return j, ok, err
		}
		select {
		case <-ctx.Done():
			return job.Job{}, false, ctx.Err()
		case <-prazo.C:
			return job.Job{}, false, nil
		case <-mudou:
			// algo mudou: volta ao Fetch
		}
	}
}

// livro:fim longpoll
