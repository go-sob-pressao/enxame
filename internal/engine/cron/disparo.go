package cron

import (
	"context"
	"log/slog"
	"time"
	"uuid"

	"github.com/go-sob-pressao/enxame/internal/core/id"
	"github.com/go-sob-pressao/enxame/internal/core/job"
	"github.com/go-sob-pressao/enxame/internal/store"
)

// livro:inicio disparo

// Disparo decide o job da janela vencida de um agendamento e a próxima
// janela. A chave única do job é a janela prevista — não a hora em que
// o disparo aconteceu —, então disparar a mesma janela duas vezes
// produz a mesma chave, e o banco guarda um job só. Janelas perdidas
// enquanto ninguém disparava são puladas: a próxima é a primeira
// depois de agora.
func Disparo(
	sc store.Schedule,
	now time.Time,
) (job.Spec, time.Time, error) {
	e, err := Parse(sc.Expr)
	if err != nil {
		return job.Spec{}, time.Time{}, err
	}
	loc, err := time.LoadLocation(sc.Timezone)
	if err != nil {
		return job.Spec{}, time.Time{}, err
	}
	spec := job.Spec{
		ID:        id.JobID(uuid.NewV7()),
		Namespace: sc.Namespace, Queue: sc.Queue, Kind: sc.Kind,
		Args:      sc.Args,
		UniqueKey: id.WindowKey(sc.Namespace, sc.ID, sc.NextFire),
	}
	return spec, e.Next(later(sc.NextFire, now), loc), nil
}

// livro:fim disparo

func later(a, b time.Time) time.Time {
	if a.After(b) {
		return a
	}
	return b
}

// Store é o que o agendador precisa do armazenamento.
type Store interface {
	FireDue(
		ctx context.Context,
		now time.Time,
		decidir func(store.Schedule, time.Time) (job.Spec, time.Time,
			error),
	) (int, int, error)
}

// Scheduler dispara os agendamentos vencidos a cada Every.
type Scheduler struct {
	Store Store
	Now   func() time.Time
	Every time.Duration
	Log   *slog.Logger
}

// Run dispara até o contexto terminar.
func (s *Scheduler) Run(ctx context.Context) error {
	t := time.NewTicker(s.Every)
	defer t.Stop()
	for {
		d, a, err := s.Store.FireDue(ctx, s.Now(), Disparo)
		if err != nil {
			s.Log.Error("disparo de agendamentos", "erro", err)
		} else if d+a > 0 {
			s.Log.Info("agendamentos", "disparados", d, "absorvidos", a)
		}
		select {
		case <-t.C:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}
