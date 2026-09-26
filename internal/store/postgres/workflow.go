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
