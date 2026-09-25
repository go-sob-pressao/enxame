package postgres

import (
	"context"
	"encoding/json"

	"github.com/jackc/pgx/v5"

	"github.com/go-sob-pressao/enxame/internal/core/id"
	"github.com/go-sob-pressao/enxame/internal/core/job"
	"github.com/go-sob-pressao/enxame/internal/store"
)

// livro:inicio seq

// acrescentar grava os eventos do job, cada um com o seq tirado de
// job.next_seq na mesma instrução que o incrementa. O UPDATE trava a
// linha do job até o COMMIT: outra transação que queira acrescentar
// eventos ao mesmo job espera, e recebe o seq seguinte ao último
// gravado — nunca um seq que ficaria para trás, nem uma lacuna se esta
// transação desistir. As instruções vão ao banco num lote só.
func acrescentar(
	ctx context.Context,
	tx pgx.Tx,
	jid id.JobID,
	evs []job.Event,
) error {
	lote := &pgx.Batch{}
	for _, e := range evs {
		p, err := json.Marshal(store.PayloadOf(e))
		if err != nil {
			return err
		}
		lote.Queue(`WITH s AS (
				UPDATE job SET next_seq = next_seq + 1 WHERE job_id = $1
				RETURNING next_seq - 1 AS seq)
			INSERT INTO job_event (job_id, seq, event_type, occurred_at,
				payload)
			SELECT $1, seq, $2, $3, $4 FROM s`,
			jid.String(), int16(e.Type), e.At, string(p))
	}
	return tx.SendBatch(ctx, lote).Close()
}

// livro:fim seq
