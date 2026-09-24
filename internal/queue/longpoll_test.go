package queue_test

import (
	"context"
	"errors"
	"testing"
	"testing/synctest"
	"time"

	"github.com/go-sob-pressao/enxame/internal/core/job"
	"github.com/go-sob-pressao/enxame/internal/queue"
)

func pollCfg() queue.PollConfig {
	return queue.PollConfig{
		Queue:       "q",
		Worker:      "w",
		PollTimeout: 30 * time.Second,
		Now:         time.Now,
	}
}

// livro:inicio longpoll-vias

// Via 1 — cancelamento: o desligamento não espera os 30 s do poll.
func TestPollCancelamentoRetornaNaHora(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		m := queue.NewMemory()
		ctx, cancel := context.WithCancel(context.Background())
		inicio := time.Now()
		go func() {
			// relógio virtual da bolha
			time.Sleep(time.Second) //nolint:forbidigo
			cancel()
		}()
		_, _, err := queue.Poll(ctx, m, m, pollCfg())
		if !errors.Is(err, context.Canceled) ||
			time.Since(inicio) != time.Second {
			t.Fatalf(
				"err=%v depois de %v; esperado Canceled em 1s",
				err,
				time.Since(inicio),
			)
		}
	})
}

// Via 2 — prazo: sem trabalho, o worker volta depois de PollTimeout.
func TestPollPrazoDevolveVazio(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		m := queue.NewMemory()
		inicio := time.Now()
		_, ok, err := queue.Poll(context.Background(), m, m, pollCfg())
		if ok || err != nil || time.Since(inicio) != 30*time.Second {
			t.Fatalf(
				"ok=%v err=%v depois de %v",
				ok,
				err,
				time.Since(inicio),
			)
		}
	})
}

// Via 3 — notificação: o job inserido começa na hora, não 30 s depois.
func TestPollNotificacaoAcordaNaHora(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		m := queue.NewMemory()
		go func() {
			// relógio virtual da bolha
			time.Sleep(2 * time.Second) //nolint:forbidigo
			_, _ = m.Insert(
				job.Spec{ID: jid(1), Queue: "q", Kind: "eco"},
				time.Now(),
			)
		}()
		inicio := time.Now()
		j, ok, err := queue.Poll(context.Background(), m, m, pollCfg())
		if !ok || err != nil || j.ID != jid(1) ||
			time.Since(inicio) != 2*time.Second {
			t.Fatalf(
				"ok=%v err=%v depois de %v",
				ok,
				err,
				time.Since(inicio),
			)
		}
	})
}

// livro:fim longpoll-vias
