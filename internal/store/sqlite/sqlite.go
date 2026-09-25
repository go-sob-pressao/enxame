package sqlite

import (
	"context"
	"database/sql"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"modernc.org/sqlite"
	sqlite3 "modernc.org/sqlite/lib"

	"github.com/go-sob-pressao/enxame/internal/core/id"
	"github.com/go-sob-pressao/enxame/internal/core/job"
	"github.com/go-sob-pressao/enxame/internal/store"
)

//go:embed schema.sql
var schema string

// Store guarda jobs num arquivo SQLite.
type Store struct{ db *sql.DB }

// Open abre (ou cria) o banco em path e aplica o esquema.
func Open(ctx context.Context, path string) (*Store, error) {
	db, err := sql.Open("sqlite", "file:"+path+
		"?_pragma=foreign_keys(1)&_pragma=busy_timeout(5000)")
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1) // um escritor por vez, como o SQLite exige
	if _, err := db.ExecContext(ctx, schema); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("aplicar esquema: %w", err)
	}
	return &Store{db: db}, nil
}

// Close fecha o banco.
func (s *Store) Close() error { return s.db.Close() }

const colunas = `job_id, namespace, queue, kind, args, unique_key,
	state, priority, attempt, max_attempts, scheduled_at, attempted_at,
	attempted_by, finalized_at, last_error, version`

// Insert grava o job e o histórico numa transação.
func (s *Store) Insert(
	ctx context.Context,
	j job.Job,
	evs []job.Event,
) error {
	return s.tx(ctx, func(tx *sql.Tx) error {
		_, err := tx.ExecContext(
			ctx,
			`INSERT INTO job (`+colunas+`)
			VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,1)`,
			j.ID.String(),
			j.Namespace,
			j.Queue,
			j.Kind,
			args(j.Args),
			nulo(j.UniqueKey),
			string(j.State),
			j.Priority,
			j.Attempt,
			j.MaxAttempts,
			nanos(j.ScheduledAt),
			nanosNulo(j.AttemptedAt),
			nulo(
				j.AttemptedBy,
			),
			nanosNulo(j.FinalizedAt),
			nulo(j.LastError),
		)
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
	row := s.db.QueryRowContext(ctx,
		`SELECT `+colunas+` FROM job WHERE job_id = ?`, jid.String())
	return ler(row)
}

// History devolve os eventos em ordem.
func (s *Store) History(
	ctx context.Context,
	jid id.JobID,
) ([]job.Event, error) {
	if _, _, err := s.Get(ctx, jid); err != nil {
		return nil, err
	}
	rows, err := s.db.QueryContext(ctx, `SELECT event_type, occurred_at,
		payload FROM job_event WHERE job_id = ? ORDER BY seq`,
		jid.String())
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var evs []job.Event
	for rows.Next() {
		var tipo, em int64
		var p []byte
		if err := rows.Scan(&tipo, &em, &p); err != nil {
			return nil, err
		}
		ev, err := evento(tipo, em, p)
		if err != nil {
			return nil, err
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
	return s.tx(ctx, func(tx *sql.Tx) error {
		res, err := tx.ExecContext(ctx, `UPDATE job SET state = ?,
			attempt = ?, scheduled_at = ?, attempted_at = ?,
			attempted_by = ?, finalized_at = ?, last_error = ?,
			version = version + 1
			WHERE job_id = ? AND version = ?`,
			string(j.State), j.Attempt, nanos(j.ScheduledAt),
			nanosNulo(j.AttemptedAt), nulo(j.AttemptedBy),
			nanosNulo(j.FinalizedAt), nulo(j.LastError),
			j.ID.String(), version)
		if err != nil {
			return traduzir(err)
		}
		if n, _ := res.RowsAffected(); n == 0 {
			var v int64
			err := tx.QueryRowContext(ctx,
				`SELECT version FROM job WHERE job_id = ?`,
				j.ID.String()).Scan(&v)
			if errors.Is(err, sql.ErrNoRows) {
				return store.ErrNotFound
			}
			return fmt.Errorf("%w: gravada %d, esperada %d",
				store.ErrConflict, v, version)
		}
		return acrescentar(ctx, tx, j.ID, evs)
	})
}

// Next devolve o próximo disponível da fila, na ordem do índice.
func (s *Store) Next(
	ctx context.Context,
	queue string,
) (job.Job, int64, bool, error) {
	row := s.db.QueryRowContext(ctx, `SELECT `+colunas+` FROM job
		WHERE queue = ? AND state = 'available'
		ORDER BY priority, scheduled_at, job_id LIMIT 1`, queue)
	j, v, err := ler(row)
	if errors.Is(err, store.ErrNotFound) {
		return job.Job{}, 0, false, nil
	}
	return j, v, err == nil, err
}

func (s *Store) tx(ctx context.Context, f func(*sql.Tx) error) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	if err := f(tx); err != nil {
		_ = tx.Rollback()
		return err
	}
	return tx.Commit()
}

// acrescentar grava os eventos com seq a partir de job.next_seq.
func acrescentar(
	ctx context.Context,
	tx *sql.Tx,
	jid id.JobID,
	evs []job.Event,
) error {
	for _, e := range evs {
		p, err := json.Marshal(store.PayloadOf(e))
		if err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO job_event
			(job_id, seq, event_type, occurred_at, payload)
			SELECT job_id, next_seq, ?, ?, ? FROM job WHERE job_id = ?`,
			int64(e.Type), nanos(e.At), string(p),
			jid.String()); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `UPDATE job
			SET next_seq = next_seq + 1 WHERE job_id = ?`,
			jid.String()); err != nil {
			return err
		}
	}
	return nil
}

type linha interface{ Scan(dest ...any) error }

func ler(r linha) (job.Job, int64, error) {
	var (
		j                   job.Job
		jid, estado         string
		args                string
		chave, por, erro    sql.NullString
		agendado            int64
		tentado, finalizado sql.NullInt64
		versao              int64
	)
	err := r.Scan(&jid, &j.Namespace, &j.Queue, &j.Kind, &args, &chave,
		&estado, &j.Priority, &j.Attempt, &j.MaxAttempts, &agendado,
		&tentado, &por, &finalizado, &erro, &versao)
	if errors.Is(err, sql.ErrNoRows) {
		return job.Job{}, 0, store.ErrNotFound
	}
	if err != nil {
		return job.Job{}, 0, err
	}
	if j.ID, err = id.ParseJobID(jid); err != nil {
		return job.Job{}, 0, err
	}
	j.Args = []byte(args)
	j.UniqueKey, j.AttemptedBy, j.LastError = chave.String, por.String,
		erro.String
	j.State = job.State(estado)
	j.ScheduledAt = time.Unix(0, agendado).UTC()
	j.AttemptedAt = instante(tentado)
	j.FinalizedAt = instante(finalizado)
	return j, versao, nil
}

// traduzir converte violações de unicidade em store.ErrDuplicate.
func traduzir(err error) error {
	var e *sqlite.Error
	if errors.As(err, &e) &&
		(e.Code() == sqlite3.SQLITE_CONSTRAINT_UNIQUE ||
			e.Code() == sqlite3.SQLITE_CONSTRAINT_PRIMARYKEY) {
		return fmt.Errorf("%w: %w", store.ErrDuplicate, err)
	}
	return err
}

func args(a []byte) string {
	if len(a) == 0 {
		return "{}"
	}
	return string(a)
}

func nulo(s string) sql.NullString {
	return sql.NullString{String: s, Valid: s != ""}
}

func nanos(t time.Time) int64 { return t.UnixNano() }

func nanosNulo(t time.Time) sql.NullInt64 {
	return sql.NullInt64{Int64: t.UnixNano(), Valid: !t.IsZero()}
}

func instante(n sql.NullInt64) time.Time {
	if !n.Valid {
		return time.Time{}
	}
	return time.Unix(0, n.Int64).UTC()
}

func evento(tipo, em int64, p []byte) (job.Event, error) {
	var pl store.Payload
	if err := json.Unmarshal(p, &pl); err != nil {
		return job.Event{}, err
	}
	ev := pl.Event(job.EventType(tipo), time.Unix(0, em))
	if pl.RunAt.IsZero() {
		ev.RunAt = time.Time{}
	}
	return ev, nil
}
