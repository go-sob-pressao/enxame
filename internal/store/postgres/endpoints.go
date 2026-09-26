package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/go-sob-pressao/enxame/internal/core/webhook"
	"github.com/go-sob-pressao/enxame/internal/store"
)

const colunasEndpoint = `endpoint_id::text, namespace, url, description,
	event_types, secret_ref, disabled_at IS NOT NULL, created_at`

func lerEndpoint(r pgx.Row) (webhook.Endpoint, error) {
	var e webhook.Endpoint
	err := r.Scan(&e.ID, &e.Namespace, &e.URL, &e.Description,
		&e.EventTypes, &e.SecretRef, &e.Disabled, &e.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return e, store.ErrNotFound
	}
	return e, err
}

// CreateEndpoint grava o endpoint e devolve-o com id e data.
func (s *Store) CreateEndpoint(
	ctx context.Context,
	e webhook.Endpoint,
) (webhook.Endpoint, error) {
	if e.EventTypes == nil {
		e.EventTypes = []string{}
	}
	return lerEndpoint(
		s.pool.QueryRow(
			ctx,
			`INSERT INTO webhook_endpoint
		(namespace, url, description, event_types, secret_ref)
		VALUES ($1, $2, $3, $4, $5) RETURNING `+colunasEndpoint,
			e.Namespace,
			e.URL,
			e.Description,
			e.EventTypes,
			e.SecretRef,
		),
	)
}

// ListEndpoints devolve os endpoints do namespace, os mais novos antes.
func (s *Store) ListEndpoints(
	ctx context.Context,
	namespace string,
) ([]webhook.Endpoint, error) {
	rows, err := s.pool.Query(ctx, `SELECT `+colunasEndpoint+`
		FROM webhook_endpoint WHERE namespace = $1
		ORDER BY created_at DESC`, namespace)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows,
		func(r pgx.CollectableRow) (webhook.Endpoint, error) {
			return lerEndpoint(r)
		})
}

// DisableEndpoint desativa o endpoint, com o motivo.
func (s *Store) DisableEndpoint(
	ctx context.Context,
	namespace, id, motivo string,
) error {
	tag, err := s.pool.Exec(ctx, `UPDATE webhook_endpoint
		SET disabled_at = now(), disabled_reason = $3
		WHERE namespace = $1 AND endpoint_id = $2::uuid
		  AND disabled_at IS NULL`, namespace, id, motivo)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("%w: endpoint %s inexistente ou já desativado",
			store.ErrNotFound, id)
	}
	return nil
}
