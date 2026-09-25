package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/go-sob-pressao/enxame/internal/core/id"
	"github.com/go-sob-pressao/enxame/internal/core/job"
	"github.com/go-sob-pressao/enxame/internal/store"
)

// Store guarda jobs no PostgreSQL, no esquema das migrações 0002 e
// 0003. Por enquanto, tudo na partição 0 (particionamento: Parte V).
type Store struct{ pool *pgxpool.Pool }

// New usa o pool dado; quem o criou é quem o fecha.
func New(pool *pgxpool.Pool) *Store { return &Store{pool: pool} }

// colunas lê a projeção; last_error vem do histórico, que é a fonte
// da verdade: a causa da última tentativa que falhou.
const colunas = `job_id::text, namespace, queue, kind, args::text,
	unique_key, state, priority, attempt, max_attempts, scheduled_at,
	attempted_at, attempted_by, finalized_at,
	(SELECT e.payload->>'cause' FROM job_event e
	  WHERE e.job_id = job.job_id AND e.event_type = 6
	  ORDER BY e.seq DESC LIMIT 1),
	version`

// Insert grava o job e o histórico numa transação.
func (s *Store) Insert(
	ctx context.Context,
	j job.Job,
	evs []job.Event,
) error {
	return pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, `INSERT INTO job (job_id, partition_id,
			namespace, queue, kind, args, unique_key, state, priority,
			attempt, max_attempts, scheduled_at, attempted_at,
			attempted_by, finalized_at, version)
			VALUES ($1, 0, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11,
			        $12, $13, $14, 1)`,
			j.ID.String(), j.Namespace, j.Queue, j.Kind, args(j.Args),
			nulo(j.UniqueKey), string(j.State), j.Priority, j.Attempt,
			j.MaxAttempts, j.ScheduledAt, instanteNulo(j.AttemptedAt),
			nulo(j.AttemptedBy), instanteNulo(j.FinalizedAt))
		if err != nil {
			return traduzir(err)
		}
		return acrescentar(ctx, tx, j.ID, evs)
	})
}

// Get devolve a projeção e a versão.
func (s *Store) Get(
	ctx context.Context,
	jid id.JobID,
) (job.Job, int64, error) {
	return ler(s.pool.QueryRow(ctx,
		`SELECT `+colunas+` FROM job WHERE job_id = $1`, jid.String()))
}

// History devolve os eventos em ordem.
func (s *Store) History(
	ctx context.Context,
	jid id.JobID,
) ([]job.Event, error) {
	if _, _, err := s.Get(ctx, jid); err != nil {
		return nil, err
	}
	rows, err := s.pool.Query(ctx, `SELECT event_type, occurred_at,
		payload::text FROM job_event WHERE job_id = $1 ORDER BY seq`,
		jid.String())
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var evs []job.Event
	for rows.Next() {
		var tipo int16
		var em time.Time
		var p string
		if err := rows.Scan(&tipo, &em, &p); err != nil {
			return nil, err
		}
		var pl store.Payload
		if err := json.Unmarshal([]byte(p), &pl); err != nil {
			return nil, err
		}
		ev := pl.Event(job.EventType(tipo), em)
		if pl.RunAt.IsZero() {
			ev.RunAt = time.Time{}
		}
		evs = append(evs, ev)
	}
	return evs, rows.Err()
}

// Update grava a projeção se a versão ainda for version.
func (s *Store) Update(
	ctx context.Context,
	j job.Job,
	evs []job.Event,
	version int64,
) error {
	return pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		tag, err := tx.Exec(
			ctx,
			`UPDATE job SET state = $2, attempt = $3,
			scheduled_at = $4, attempted_at = $5, attempted_by = $6,
			finalized_at = $7, version = version + 1
			WHERE job_id = $1 AND version = $8`,
			j.ID.String(),
			string(j.State),
			j.Attempt,
			j.ScheduledAt,
			instanteNulo(j.AttemptedAt),
			nulo(j.AttemptedBy),
			instanteNulo(j.FinalizedAt),
			version,
		)
		if err != nil {
			return traduzir(err)
		}
		if tag.RowsAffected() == 0 {
			var v int64
			err := tx.QueryRow(ctx,
				`SELECT version FROM job WHERE job_id = $1`,
				j.ID.String()).Scan(&v)
			if errors.Is(err, pgx.ErrNoRows) {
				return store.ErrNotFound
			}
			return fmt.Errorf("%w: gravada %d, esperada %d",
				store.ErrConflict, v, version)
		}
		return acrescentar(ctx, tx, j.ID, evs)
	})
}

// Next devolve o próximo disponível da fila, na ordem do índice
// job_busca.
func (s *Store) Next(
	ctx context.Context,
	queue string,
) (job.Job, int64, bool, error) {
	j, v, err := ler(s.pool.QueryRow(ctx, `SELECT `+colunas+` FROM job
		WHERE queue = $1 AND state = 'available'
		ORDER BY priority, scheduled_at, job_id LIMIT 1`, queue))
	if errors.Is(err, store.ErrNotFound) {
		return job.Job{}, 0, false, nil
	}
	return j, v, err == nil, err
}

// acrescentar grava cada evento com seq tirado de job.next_seq, na
// mesma instrução que o incrementa.
func acrescentar(
	ctx context.Context,
	tx pgx.Tx,
	jid id.JobID,
	evs []job.Event,
) error {
	for _, e := range evs {
		p, err := json.Marshal(store.PayloadOf(e))
		if err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `WITH s AS (
				UPDATE job SET next_seq = next_seq + 1 WHERE job_id = $1
				RETURNING next_seq - 1 AS seq)
			INSERT INTO job_event (job_id, seq, event_type, occurred_at,
				payload)
			SELECT $1, seq, $2, $3, $4 FROM s`,
			jid.String(), int16(e.Type), e.At, string(p)); err != nil {
			return err
		}
	}
	return nil
}

func ler(r pgx.Row) (job.Job, int64, error) {
	var (
		j                      job.Job
		jid, estado, args      string
		chave, por, ultimoErro *string
		tentado, finalizado    *time.Time
		prioridade             int16
		versao                 int64
	)
	err := r.Scan(
		&jid,
		&j.Namespace,
		&j.Queue,
		&j.Kind,
		&args,
		&chave,
		&estado,
		&prioridade,
		&j.Attempt,
		&j.MaxAttempts,
		&j.ScheduledAt,
		&tentado,
		&por,
		&finalizado,
		&ultimoErro,
		&versao,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return job.Job{}, 0, store.ErrNotFound
	}
	if err != nil {
		return job.Job{}, 0, err
	}
	if j.ID, err = id.ParseJobID(jid); err != nil {
		return job.Job{}, 0, err
	}
	j.Args = []byte(args)
	j.UniqueKey, j.AttemptedBy, j.LastError = texto(chave), texto(por),
		texto(ultimoErro)
	j.State = job.State(estado)
	j.Priority = int(prioridade)
	j.ScheduledAt = j.ScheduledAt.UTC()
	j.AttemptedAt, j.FinalizedAt = utc(tentado), utc(finalizado)
	return j, versao, nil
}

// traduzir converte a violação de unicidade (23505) em ErrDuplicate.
func traduzir(err error) error {
	var pg *pgconn.PgError
	if errors.As(err, &pg) && pg.Code == "23505" {
		return fmt.Errorf(
			"%w: %s",
			store.ErrDuplicate,
			pg.ConstraintName,
		)
	}
	return err
}

func args(a []byte) string {
	if len(a) == 0 {
		return "{}"
	}
	return string(a)
}

func nulo(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func instanteNulo(t time.Time) *time.Time {
	if t.IsZero() {
		return nil
	}
	return &t
}

func texto(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func utc(t *time.Time) time.Time {
	if t == nil {
		return time.Time{}
	}
	return t.UTC()
}
