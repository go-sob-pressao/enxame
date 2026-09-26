package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/go-sob-pressao/enxame/internal/core/job"
	nucleo "github.com/go-sob-pressao/enxame/internal/core/webhook"
	"github.com/go-sob-pressao/enxame/internal/store"
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

// Mensagem lê uma mensagem de webhook.
func (s *Store) Mensagem(
	ctx context.Context,
	id string,
) (nucleo.Mensagem, error) {
	var m nucleo.Mensagem
	var p string
	err := s.pool.QueryRow(ctx, `SELECT message_id::text, namespace,
		event_type, payload::text, created_at FROM webhook_message
		WHERE message_id = $1`, id).Scan(&m.ID, &m.Namespace,
		&m.EventType, &p, &m.CriadaEm)
	if errors.Is(err, pgx.ErrNoRows) {
		return m, store.ErrNotFound
	}
	m.Payload = []byte(p)
	return m, err
}

// Endpoint lê um endpoint.
func (s *Store) Endpoint(
	ctx context.Context,
	id string,
) (nucleo.Endpoint, error) {
	return lerEndpoint(s.pool.QueryRow(ctx, `SELECT `+colunasEndpoint+`
		FROM webhook_endpoint WHERE endpoint_id = $1`, id))
}

// RegistrarTentativa grava uma tentativa de entrega.
func (s *Store) RegistrarTentativa(
	ctx context.Context,
	t nucleo.Tentativa,
) error {
	_, err := s.pool.Exec(ctx, `INSERT INTO webhook_attempt
		(message_id, endpoint_id, attempt, job_id, status_code,
		 duration_ms, error, response_excerpt, attempted_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT DO NOTHING`, t.MessageID, t.EndpointID, t.Attempt,
		t.JobID, nuloInt(t.Status), t.Duracao.Milliseconds(),
		nulo(t.Erro), nulo(t.Trecho), t.Em)
	return err
}

// livro:inicio fanout

// EnfileirarEntregas cria, numa transação, um job de entrega por
// endpoint. A chave única por mensagem e endpoint torna o fan-out
// idempotente: se ele cair no meio e for repetido, os jobs que já
// existiam são absorvidos, e só os que faltavam são criados.
func (s *Store) EnfileirarEntregas(
	ctx context.Context,
	m nucleo.Mensagem,
	endpoints []string,
	args func(endpoint string) []byte,
) (int, error) {
	criados := 0
	err := pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		for _, e := range endpoints {
			err := pgx.BeginFunc(ctx, tx, func(sp pgx.Tx) error {
				_, err := enfileirar(ctx, sp, job.Spec{
					Namespace: m.Namespace, Queue: webhook.FanoutQueue,
					Kind: webhook.DeliverKind, Args: args(e),
					UniqueKey: "entrega:" + m.ID + ":" + e,
				}, time.Now())
				return err
			})
			switch {
			case errors.Is(err, store.ErrDuplicate):
			case err != nil:
				return err
			default:
				criados++
			}
		}
		return nil
	})
	return criados, err
}

// livro:fim fanout

func nuloInt(n int) *int {
	if n == 0 {
		return nil
	}
	return &n
}
