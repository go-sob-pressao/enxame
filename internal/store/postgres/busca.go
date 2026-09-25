package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/go-sob-pressao/enxame/internal/core/id"
	"github.com/go-sob-pressao/enxame/internal/core/job"
	"github.com/go-sob-pressao/enxame/internal/store"
)

// livro:inicio busca

// Claim reserva o próximo job disponível da fila para worker, numa
// transação: trava a linha com FOR UPDATE SKIP LOCKED — outros workers,
// ao mesmo tempo, pulam a linha travada e pegam a seguinte, em vez de
// esperar por ela —, decide pelo domínio e grava a projeção e o evento.
func (s *Store) Claim(
	ctx context.Context,
	queue string,
	at time.Time,
	worker string,
) (job.Job, bool, error) {
	var reservado job.Job
	achou := false
	err := pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		j, v, err := ler(tx.QueryRow(ctx, `SELECT `+colunas+` FROM job
			WHERE partition_id = 0 AND queue = $1
			  AND state = 'available'
			ORDER BY priority, scheduled_at, job_id
			LIMIT 1 FOR UPDATE SKIP LOCKED`, queue))
		if errors.Is(err, store.ErrNotFound) {
			return nil // fila vazia, ou tudo travado por outros
		}
		if err != nil {
			return err
		}
		evs, err := job.Start(j, at, worker)
		if err != nil {
			return err
		}
		if reservado, err = job.ApplyAll(j, evs); err != nil {
			return err
		}
		achou = true
		return gravar(ctx, tx, reservado, evs, v)
	})
	return reservado, achou, err
}

// livro:fim busca

// Decide aplica, numa transação e com a linha travada, uma decisão do
// domínio ao job jid: Complete, Fail, Cancel, Rescue.
func (s *Store) Decide(
	ctx context.Context,
	jid id.JobID,
	decidir func(job.Job) ([]job.Event, error),
) (job.Job, error) {
	var novo job.Job
	err := pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		j, v, err := ler(tx.QueryRow(ctx,
			`SELECT `+colunas+` FROM job WHERE job_id = $1 FOR UPDATE`,
			jid.String()))
		if err != nil {
			return err
		}
		evs, err := decidir(j)
		if err != nil {
			return err
		}
		if novo, err = job.ApplyAll(j, evs); err != nil {
			return err
		}
		return gravar(ctx, tx, novo, evs, v)
	})
	return novo, err
}

// Promote torna disponíveis, em lotes de até 100, os jobs agendados ou
// à espera de retry cuja hora chegou. Devolve quantos promoveu.
func (s *Store) Promote(
	ctx context.Context,
	at time.Time,
) (int, error) {
	return s.emLote(ctx, `partition_id = 0
		AND state IN ('scheduled', 'retryable')
		AND scheduled_at <= $1`, at,
		func(j job.Job) ([]job.Event, error) {
			return job.MakeAvailable(j, at)
		})
}

// livro:inicio resgate

// Rescue resgata, em lotes de até 100, os jobs em execução cuja
// tentativa começou antes de desde: o processo que os reservou morreu
// sem registrar o fim. É um prazo fixo, não um batimento cardíaco —
// uma tentativa legítima mais longa que o prazo seria resgatada viva.
// O lease de verdade é da Parte V.
func (s *Store) Rescue(
	ctx context.Context,
	at, desde time.Time,
) (int, error) {
	return s.emLote(ctx, `partition_id = 0
		AND state = 'running' AND attempted_at < $1`,
		desde, func(j job.Job) ([]job.Event, error) {
			return job.Rescue(j, at)
		})
}

// livro:fim resgate

// emLote trava até 100 jobs que satisfazem onde, pulando os já
// travados, e aplica a cada um a decisão do domínio.
func (s *Store) emLote(
	ctx context.Context,
	onde string,
	arg any,
	decidir func(job.Job) ([]job.Event, error),
) (int, error) {
	n := 0
	err := pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, `SELECT `+colunas+` FROM job
			WHERE `+onde+` ORDER BY job_id LIMIT 100
			FOR UPDATE SKIP LOCKED`, arg)
		if err != nil {
			return err
		}
		type travado struct {
			j job.Job
			v int64
		}
		var lote []travado
		for rows.Next() {
			j, v, err := ler(rows)
			if err != nil {
				rows.Close()
				return err
			}
			lote = append(lote, travado{j, v})
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			return err
		}
		for _, t := range lote {
			evs, err := decidir(t.j)
			if err != nil {
				return err
			}
			j, err := job.ApplyAll(t.j, evs)
			if err != nil {
				return err
			}
			if err := gravar(ctx, tx, j, evs, t.v); err != nil {
				return err
			}
			n++
		}
		return nil
	})
	return n, err
}

// gravar atualiza a projeção travada e acrescenta os eventos.
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
