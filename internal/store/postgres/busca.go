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

// Heartbeat registra que a tentativa attempt do job continua viva.
func (s *Store) Heartbeat(
	ctx context.Context,
	jid id.JobID,
	at time.Time,
	attempt int,
) error {
	_, err := s.Decide(ctx, jid, func(j job.Job) ([]job.Event, error) {
		return job.Heartbeat(j, at, attempt)
	})
	return err
}

// livro:inicio resgate

// Rescue resgata, em lotes de até 100, os jobs em execução cujo último
// sinal de vida — o último batimento, ou o início da tentativa — é
// anterior a desde: o worker morreu, ou perdeu a conexão, sem
// registrar o fim. Uma tentativa longa que bate a tempo não é
// resgatada.
func (s *Store) Rescue(
	ctx context.Context,
	at, desde time.Time,
) (int, error) {
	return s.emLote(ctx, `partition_id = 0 AND state = 'running'
		AND coalesce(heartbeat_at, attempted_at) < $1`,
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
