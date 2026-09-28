package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5"

	"github.com/go-sob-pressao/enxame/internal/core/id"
	"github.com/go-sob-pressao/enxame/internal/core/job"
	"github.com/go-sob-pressao/enxame/internal/observ/tracing"
	"github.com/go-sob-pressao/enxame/internal/store"
	"github.com/go-sob-pressao/enxame/pkg/workflow"
)

// livro:inicio transicao

// AppendStep grava um passo do run e, se next não for zero, o job que
// executa a posição seguinte — tudo na mesma transação. O evento (o
// passo), o estado (a posição do run) e o trabalho futuro (o job)
// entram no mesmo COMMIT: não existe o passo gravado sem o job que
// continua o run, nem o job sem o passo.
func (s *Store) AppendStep(
	ctx context.Context,
	runID string,
	r workflow.Record,
	next workflow.Continuation,
) error {
	return s.transacao(ctx, func(tx pgx.Tx) error {
		var namespace string
		err := tx.QueryRow(ctx, `UPDATE workflow_run
			SET next_step_seq = $2 + 1, version = version + 1
			WHERE run_id = $1 AND state = 'running'
			  AND next_step_seq = $2
			RETURNING namespace`, runID, r.Seq).Scan(&namespace)
		if errors.Is(err, pgx.ErrNoRows) {
			// Posição já gravada, ou run encerrado.
			return store.ErrConflict
		}
		if err != nil {
			return err
		}
		if err := gravarPasso(ctx, tx, runID, r); err != nil {
			return err
		}
		if next.At.IsZero() {
			return nil
		}
		return continuar(ctx, tx, namespace, runID, next)
	})
}

// livro:fim transicao

// continuar enfileira, na transação dada, o job que avança o run a
// partir da posição next.Seq. A chave única por posição absorve uma
// segunda tentativa de enfileirar a mesma continuação.
func continuar(
	ctx context.Context,
	tx pgx.Tx,
	namespace, runID string,
	next workflow.Continuation,
) error {
	a, err := json.Marshal(workflow.AdvanceArgs{RunID: runID,
		Seq: next.Seq})
	if err != nil {
		return err
	}
	_, err = enfileirar(ctx, tx, job.Spec{
		Namespace: namespace, Queue: workflow.AdvanceQueue,
		Kind: workflow.AdvanceKind, Args: a, RunAt: next.At,
		UniqueKey: fmt.Sprintf("workflow:%s:%d", runID, next.Seq),
	}, time.Now())
	return err
}

// enfileirar cria o job pelo domínio e o grava na transação dada. O id
// é gerado aqui, na borda.
func enfileirar(
	ctx context.Context,
	tx pgx.Tx,
	spec job.Spec,
	at time.Time,
) (job.Job, error) {
	if spec.ID.IsZero() {
		spec.ID = id.JobID(uuid.NewV7())
	}
	// O job novo continua o trace de quem o criou (Cap. 30).
	if spec.TraceParent == "" {
		spec.TraceParent = tracing.TraceParent(ctx)
	}
	evs, j, err := job.Insert(spec, at)
	if err != nil {
		return job.Job{}, err
	}
	if j, err = job.ApplyAll(j, evs); err != nil {
		return job.Job{}, err
	}
	return j, inserir(ctx, tx, j, evs)
}

func gravarPasso(
	ctx context.Context,
	tx pgx.Tx,
	runID string,
	r workflow.Record,
) error {
	estado, falha := "completed", any(nil)
	if r.Err != "" {
		estado, falha = "failed", r.Err
	}
	_, err := tx.Exec(ctx, `INSERT INTO workflow_step
		(run_id, step_seq, step_name, step_kind, state, output, error,
		 wake_at, completed_at)
		VALUES ($1, $2, $3, $4, $5, $6, to_jsonb($7::text), $8, now())`,
		runID, r.Seq, r.Name, string(r.Kind), estado,
		jsonNulo(r.Output), falha, instanteNulo(r.WakeAt))
	if errors.Is(traduzir(err), store.ErrDuplicate) {
		return store.ErrConflict
	}
	return err
}

// StartRun cria o run e o job da primeira posição, numa transação. O
// índice parcial workflow_run_aberto recusa um segundo run aberto do
// mesmo workflow_id.
func (s *Store) StartRun(
	ctx context.Context,
	run workflow.Run,
) (string, error) {
	var runID string
	err := pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		var err error
		runID, err = s.StartRunTx(ctx, tx, run)
		return err
	})
	return runID, err
}

// StartRunTx cria o run e o job da primeira posição na transação de
// quem chama.
func (s *Store) StartRunTx(
	ctx context.Context,
	tx pgx.Tx,
	run workflow.Run,
) (string, error) {
	var runID string
	err := tx.QueryRow(ctx, `INSERT INTO workflow_run
		(partition_id, namespace, workflow_id, workflow_type,
		 code_version, queue, input)
		VALUES (0, $1, $2, $3, 1, $4, $5)
		RETURNING run_id::text`,
		run.Namespace, run.WorkflowID, run.Type,
		workflow.AdvanceQueue, args(run.Input),
	).Scan(&runID)
	if err != nil {
		return "", traduzir(err)
	}
	return runID, continuar(ctx, tx, run.Namespace, runID,
		workflow.Continuation{Seq: 1, At: time.Now()})
}
