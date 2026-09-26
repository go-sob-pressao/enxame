package cron

import (
	"context"
	"log/slog"
	"time"

	"github.com/go-sob-pressao/enxame/internal/core/job"
	"github.com/go-sob-pressao/enxame/internal/core/schedule"
)

// Store é o que o agendador precisa do armazenamento.
type Store interface {
	FireDue(
		ctx context.Context,
		now time.Time,
		decidir func(schedule.Schedule, time.Time) (job.Spec, time.Time,
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
		d, a, err := s.Store.FireDue(ctx, s.Now(), schedule.Disparo)
		if err != nil {
			s.Log.ErrorContext(ctx, "disparo de agendamentos",
				slog.Any("erro", err))
		} else if d+a > 0 {
			s.Log.InfoContext(ctx, "agendamentos",
				slog.Int("disparados", d), slog.Int("absorvidos", a))
		}
		select {
		case <-t.C:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}
