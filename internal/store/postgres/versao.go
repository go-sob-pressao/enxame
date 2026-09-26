package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/go-sob-pressao/enxame/internal/core/job"
	"github.com/go-sob-pressao/enxame/internal/store"
)

// Update grava a projeção e os eventos se a versão gravada ainda for
// version: o lock otimista do contrato do Capítulo 12.
func (s *Store) Update(
	ctx context.Context,
	j job.Job,
	evs []job.Event,
	version int64,
) error {
	return pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		err := gravar(ctx, tx, j, evs, version)
		if !errors.Is(err, store.ErrConflict) {
			return err
		}
		var v int64
		err = tx.QueryRow(ctx, `SELECT version FROM job
			WHERE job_id = $1`, j.ID.String()).Scan(&v)
		if errors.Is(err, pgx.ErrNoRows) {
			return store.ErrNotFound
		}
		return fmt.Errorf("%w: gravada %d, esperada %d",
			store.ErrConflict, v, version)
	})
}

// livro:inicio versao

// gravar atualiza a projeção só se ninguém a mudou desde a leitura: a
// versão lida vai no WHERE, e o UPDATE a incrementa. Se outra transação
// gravou antes, nenhuma linha é afetada, e quem leu uma versão velha
// recebe ErrConflict — em vez de apagar, sem saber, a escrita da outra.
func gravar(
	ctx context.Context,
	tx pgx.Tx,
	j job.Job,
	evs []job.Event,
	version int64,
) error {
	tag, err := tx.Exec(ctx, `UPDATE job SET state = $2, attempt = $3,
		scheduled_at = $4, attempted_at = $5, attempted_by = $6,
		finalized_at = $7, version = version + 1
		WHERE job_id = $1 AND version = $8`,
		j.ID.String(), string(j.State), j.Attempt, j.ScheduledAt,
		instanteNulo(j.AttemptedAt), nulo(j.AttemptedBy),
		instanteNulo(j.FinalizedAt), version)
	if err != nil {
		return traduzir(err)
	}
	if tag.RowsAffected() == 0 {
		return store.ErrConflict
	}
	return acrescentar(ctx, tx, j.ID, evs)
}

// livro:fim versao
