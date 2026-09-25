package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/go-sob-pressao/enxame/internal/store"
	"github.com/go-sob-pressao/enxame/pkg/workflow"
)

// StartRun cria o run; o banco gera o id (uuidv7). O índice parcial
// workflow_run_aberto recusa um segundo run aberto do mesmo
// workflow_id.
func (s *Store) StartRun(
	ctx context.Context,
	run workflow.Run,
) (string, error) {
	var id string
	err := s.pool.QueryRow(ctx, `INSERT INTO workflow_run
		(partition_id, namespace, workflow_id, workflow_type,
		 code_version, queue, input)
		VALUES (0, $1, $2, $3, 1, 'workflow', $4)
		RETURNING run_id::text`,
		run.Namespace, run.WorkflowID, run.Type, args(run.Input),
	).Scan(&id)
	return id, traduzir(err)
}

// LoadRun devolve o run e os passos gravados, em ordem.
func (s *Store) LoadRun(
	ctx context.Context,
	runID string,
) (workflow.Run, []workflow.Record, error) {
	var (
		run          workflow.Run
		saida, falha []byte
	)
	err := s.pool.QueryRow(ctx, `SELECT run_id::text, namespace,
		workflow_id, workflow_type, input, state, output, error
		FROM workflow_run WHERE run_id = $1`, runID).Scan(
		&run.ID, &run.Namespace, &run.WorkflowID, &run.Type,
		&run.Input, &run.State, &saida, &falha)
	if errors.Is(err, pgx.ErrNoRows) {
		return run, nil, store.ErrNotFound
	}
	if err != nil {
		return run, nil, err
	}
	run.Output = saida
	if falha != nil {
		if err := json.Unmarshal(falha, &run.Err); err != nil {
			return run, nil, err
		}
	}
	rows, err := s.pool.Query(ctx, `SELECT step_seq, step_name,
		step_kind, output, error, wake_at
		FROM workflow_step WHERE run_id = $1 ORDER BY step_seq`, runID)
	if err != nil {
		return run, nil, err
	}
	passos, err := pgx.CollectRows(rows,
		func(row pgx.CollectableRow) (workflow.Record, error) {
			var (
				r      workflow.Record
				falha  []byte
				acorda *time.Time
			)
			err := row.Scan(&r.Seq, &r.Name, &r.Kind, &r.Output,
				&falha, &acorda)
			if err == nil && falha != nil {
				err = json.Unmarshal(falha, &r.Err)
			}
			if acorda != nil {
				r.WakeAt = *acorda
			}
			return r, err
		})
	return run, passos, err
}

// AppendStep grava o passo, desde que o run esteja aberto. A chave
// primária (run_id, step_seq) recusa a mesma posição gravada duas vezes
// — duas execuções do mesmo run ao mesmo tempo.
func (s *Store) AppendStep(
	ctx context.Context,
	runID string,
	r workflow.Record,
) error {
	estado, falha := "completed", any(nil)
	if r.Err != "" {
		estado, falha = "failed", r.Err
	}
	tag, err := s.pool.Exec(
		ctx,
		`INSERT INTO workflow_step
		(run_id, step_seq, step_name, step_kind, state, output, error,
		 wake_at, completed_at)
		SELECT run_id, $2, $3, $4, $5, $6, to_jsonb($7::text), $8, now()
		FROM workflow_run WHERE run_id = $1 AND state = 'running'`,
		runID,
		r.Seq,
		r.Name,
		string(r.Kind),
		estado,
		jsonNulo(r.Output),
		falha,
		instanteNulo(r.WakeAt),
	)
	if err != nil {
		if errors.Is(traduzir(err), store.ErrDuplicate) {
			return store.ErrConflict
		}
		return err
	}
	if tag.RowsAffected() == 0 {
		return store.ErrConflict
	}
	return nil
}

// CloseRun encerra o run, se ainda estiver aberto.
func (s *Store) CloseRun(ctx context.Context, run workflow.Run) error {
	var falha any
	if run.Err != "" {
		falha = run.Err
	}
	tag, err := s.pool.Exec(ctx, `UPDATE workflow_run
		SET state = $2, output = $3, error = to_jsonb($4::text),
		    closed_at = now(), version = version + 1
		WHERE run_id = $1 AND state = 'running'`,
		run.ID, string(run.State), jsonNulo(run.Output), falha)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return store.ErrConflict
	}
	return nil
}

func jsonNulo(b json.RawMessage) *string {
	if len(b) == 0 {
		return nil
	}
	s := string(b)
	return &s
}
