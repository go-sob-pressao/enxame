package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/go-sob-pressao/enxame/internal/core/job"
	"github.com/go-sob-pressao/enxame/internal/store"
)

// UpsertSchedule cria ou substitui um agendamento.
func (s *Store) UpsertSchedule(
	ctx context.Context,
	sc store.Schedule,
) error {
	_, err := s.pool.Exec(ctx, `INSERT INTO schedule (namespace,
		schedule_id, cron_expr, timezone, queue, kind, args,
		next_fire_at) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (namespace, schedule_id) DO UPDATE SET
		  cron_expr = excluded.cron_expr, timezone = excluded.timezone,
		  queue = excluded.queue, kind = excluded.kind,
		  args = excluded.args, next_fire_at = excluded.next_fire_at`,
		sc.Namespace, sc.ID, sc.Expr, sc.Timezone, sc.Queue, sc.Kind,
		args(sc.Args), sc.NextFire)
	return err
}

// livro:inicio fire-due

// FireDue dispara os agendamentos vencidos até now, numa transação:
// trava cada um com SKIP LOCKED, enfileira o job da janela e avança
// next_fire_at. Um job com a mesma chave — a janela já disparada — é
// absorvido pelo índice único, e o agendamento avança assim mesmo.
func (s *Store) FireDue(
	ctx context.Context,
	now time.Time,
	decidir func(store.Schedule, time.Time) (
		job.Spec, time.Time, error),
) (disparados, absorvidos int, err error) {
	err = pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		vencidos, err := vencidos(ctx, tx, now)
		if err != nil {
			return err
		}
		for _, sc := range vencidos {
			spec, proxima, err := decidir(sc, now)
			if err != nil {
				return err
			}
			// Um savepoint: a duplicata desfaz só o INSERT do job.
			err = pgx.BeginFunc(ctx, tx, func(sp pgx.Tx) error {
				_, err := enfileirar(ctx, sp, spec, now)
				return err
			})
			switch {
			case errors.Is(err, store.ErrDuplicate):
				absorvidos++
			case err != nil:
				return err
			default:
				disparados++
			}
			if _, err := tx.Exec(ctx, `UPDATE schedule
				SET next_fire_at = $3, last_fired_at = $4
				WHERE namespace = $1 AND schedule_id = $2`,
				sc.Namespace, sc.ID, proxima, now); err != nil {
				return err
			}
		}
		return nil
	})
	return disparados, absorvidos, err
}

// livro:fim fire-due

func vencidos(
	ctx context.Context,
	tx pgx.Tx,
	now time.Time,
) ([]store.Schedule, error) {
	rows, err := tx.Query(ctx, `SELECT namespace, schedule_id,
		cron_expr, timezone, queue, kind, args::text, next_fire_at
		FROM schedule WHERE NOT paused AND next_fire_at <= $1
		ORDER BY next_fire_at LIMIT 100 FOR UPDATE SKIP LOCKED`, now)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows,
		func(r pgx.CollectableRow) (store.Schedule, error) {
			var sc store.Schedule
			var a string
			err := r.Scan(&sc.Namespace, &sc.ID, &sc.Expr, &sc.Timezone,
				&sc.Queue, &sc.Kind, &a, &sc.NextFire)
			sc.Args = []byte(a)
			return sc, err
		})
}
