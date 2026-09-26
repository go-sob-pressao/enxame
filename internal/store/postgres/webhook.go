package postgres

import (
	"context"
	"encoding/json"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/go-sob-pressao/enxame/internal/core/job"
	"github.com/go-sob-pressao/enxame/pkg/webhook"
)

// PublishTx grava a mensagem e o job que a distribui aos endpoints, na
// transação de quem chama: a tabela webhook_message é o outbox.
func (s *Store) PublishTx(
	ctx context.Context,
	tx pgx.Tx,
	namespace string,
	m webhook.Message,
) (string, error) {
	p, err := json.Marshal(m.Payload)
	if err != nil {
		return "", err
	}
	var id string
	if err := tx.QueryRow(ctx, `INSERT INTO webhook_message
		(namespace, event_type, payload, idempotency_key)
		VALUES ($1, $2, $3, $4) RETURNING message_id::text`,
		namespace, m.EventType, string(p), nulo(m.IdempotencyKey),
	).Scan(&id); err != nil {
		return "", traduzir(err)
	}
	a, err := json.Marshal(map[string]string{"message_id": id})
	if err != nil {
		return "", err
	}
	_, err = enfileirar(ctx, tx, job.Spec{
		Namespace: namespace, Queue: webhook.FanoutQueue,
		Kind: webhook.FanoutKind, Args: a,
	}, time.Now())
	return id, err
}
