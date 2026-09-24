package queue

import (
	"context"
	"time"

	"github.com/go-sob-pressao/enxame/internal/core/job"
)

// Source é a fila vista por quem só busca trabalho.
type Source interface {
	Fetch(
		queue string,
		at time.Time,
		worker string,
	) (job.Job, bool, error)
}

// FeedConfig configura o Feed.
type FeedConfig struct {
	Queue  string
	Worker string
	Idle   time.Duration // espera quando a fila está vazia
	Now    func() time.Time
}

// livro:inicio feed

// Feed entrega os jobs da fila por um canal. Uma única goroutine busca
// e envia — e só ela fecha o canal, quando o contexto é cancelado ou a
// busca falha. Quem recebe vê um <-chan: não pode enviar nem fechar, e
// o compilador garante isso.
func Feed(
	ctx context.Context,
	src Source,
	cfg FeedConfig,
) <-chan job.Job {
	// sem buffer: o Feed só busca o próximo quando alguém recebeu
	saida := make(chan job.Job)
	go func() {
		defer close(saida)
		for {
			j, ok, err := src.Fetch(cfg.Queue, cfg.Now(), cfg.Worker)
			if err != nil {
				return
			}
			if !ok {
				select {
				case <-time.After(cfg.Idle):
					continue
				case <-ctx.Done():
					return
				}
			}
			select {
			case saida <- j:
			case <-ctx.Done():
				return
			}
		}
	}()
	return saida
}

// livro:fim feed
