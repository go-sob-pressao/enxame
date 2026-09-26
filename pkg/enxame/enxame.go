package enxame

import (
	"context"
	"encoding/json"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/go-sob-pressao/enxame/internal/core/id"
	"github.com/go-sob-pressao/enxame/internal/core/job"
	"github.com/go-sob-pressao/enxame/internal/store"
	"github.com/go-sob-pressao/enxame/internal/store/postgres"
	"github.com/go-sob-pressao/enxame/pkg/webhook"
	"github.com/go-sob-pressao/enxame/pkg/workflow"
)

// ErrDuplicate indica que já existe, no namespace, um job com a mesma
// chave única.
var ErrDuplicate = store.ErrDuplicate

// Args são os argumentos tipados de um job; Kind escolhe o handler.
type Args interface{ Kind() string }

// Client enfileira jobs, inicia workflows e publica mensagens no
// Postgres da aplicação.
type Client struct {
	store     *postgres.Store
	pool      *pgxpool.Pool
	namespace string
}

// New cria o cliente sobre o pool da aplicação, num namespace.
func New(pool *pgxpool.Pool, namespace string) *Client {
	return &Client{store: postgres.New(pool), pool: pool,
		namespace: namespace}
}

// Option ajusta um enfileiramento.
type Option func(*job.Spec)

// Queue escolhe a fila; o padrão é "default".
func Queue(q string) Option { return func(s *job.Spec) { s.Queue = q } }

// UniqueKey impede um segundo job com a mesma chave no namespace.
func UniqueKey(k string) Option {
	return func(s *job.Spec) { s.UniqueKey = k }
}

// RunAt agenda o job para o instante dado.
func RunAt(t time.Time) Option {
	return func(s *job.Spec) { s.RunAt = t }
}

// livro:inicio insert-tx

// Insert enfileira o job numa transação própria.
func (c *Client) Insert(
	ctx context.Context,
	a Args,
	opts ...Option,
) (string, error) {
	var jid string
	err := pgx.BeginFunc(ctx, c.pool, func(tx pgx.Tx) error {
		var err error
		jid, err = c.InsertTx(ctx, tx, a, opts...)
		return err
	})
	return jid, err
}

// InsertTx enfileira o job na transação da aplicação. O job passa a
// existir no COMMIT dela, junto com os dados de negócio, e desaparece
// no ROLLBACK: não há janela em que um exista sem o outro.
func (c *Client) InsertTx(
	ctx context.Context,
	tx pgx.Tx,
	a Args,
	opts ...Option,
) (string, error) {
	args, err := json.Marshal(a)
	if err != nil {
		return "", err
	}
	spec := job.Spec{ID: id.JobID(uuid.NewV7()),
		Namespace: c.namespace, Queue: "default", Kind: a.Kind(),
		Args: args}
	for _, o := range opts {
		o(&spec)
	}
	evs, j, err := job.Insert(spec, time.Now())
	if err != nil {
		return "", err
	}
	if j, err = job.ApplyAll(j, evs); err != nil {
		return "", err
	}
	return j.ID.String(), c.store.InsertTx(ctx, tx, j, evs)
}

// PublishTx grava a mensagem de webhook na transação da aplicação.
func (c *Client) PublishTx(
	ctx context.Context,
	tx pgx.Tx,
	m webhook.Message,
) (string, error) {
	return c.store.PublishTx(ctx, tx, c.namespace, m)
}

// livro:fim insert-tx

// StartWorkflowTx inicia um run na transação da aplicação: o run e o
// job da primeira posição existem no COMMIT dela.
func (c *Client) StartWorkflowTx(
	ctx context.Context,
	tx pgx.Tx,
	tipo, workflowID string,
	input any,
) (string, error) {
	in, err := json.Marshal(input)
	if err != nil {
		return "", err
	}
	return c.store.StartRunTx(ctx, tx, workflow.Run{
		Namespace: c.namespace, WorkflowID: workflowID, Type: tipo,
		Input: in,
	})
}
