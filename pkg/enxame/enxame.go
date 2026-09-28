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
	"github.com/go-sob-pressao/enxame/internal/core/schedule"
	nucleo "github.com/go-sob-pressao/enxame/internal/core/webhook"
	"github.com/go-sob-pressao/enxame/internal/observ/tracing"
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

// OrderingKey faz os jobs com a mesma chave rodarem um por vez, na
// ordem em que foram enfileirados — e todos na mesma partição.
func OrderingKey(k string) Option {
	return func(s *job.Spec) { s.OrderingKey = k }
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
		Args: args, TraceParent: tracing.TraceParent(ctx)}
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

// StartWorkflow inicia um run numa transação própria.
func (c *Client) StartWorkflow(
	ctx context.Context,
	tipo, workflowID string,
	input any,
) (string, error) {
	var runID string
	err := pgx.BeginFunc(ctx, c.pool, func(tx pgx.Tx) error {
		var err error
		runID, err = c.StartWorkflowTx(ctx, tx, tipo, workflowID, input)
		return err
	})
	return runID, err
}

// Schedule é um agendamento periódico.
type Schedule struct {
	ID       string
	Cron     string // cinco campos: minuto hora dia mês dia-da-semana
	Timezone string // "UTC" se vazio
	Queue    string // "default" se vazio
	Args     Args
}

// Schedule cria ou substitui um agendamento; a primeira janela é a
// próxima depois de agora.
func (c *Client) Schedule(ctx context.Context, s Schedule) error {
	if s.Timezone == "" {
		s.Timezone = "UTC"
	}
	if s.Queue == "" {
		s.Queue = "default"
	}
	e, err := schedule.Parse(s.Cron)
	if err != nil {
		return err
	}
	loc, err := time.LoadLocation(s.Timezone)
	if err != nil {
		return err
	}
	a, err := json.Marshal(s.Args)
	if err != nil {
		return err
	}
	return c.store.UpsertSchedule(ctx, schedule.Schedule{
		Namespace: c.namespace, ID: s.ID, Expr: s.Cron,
		Timezone: s.Timezone, Queue: s.Queue, Kind: s.Args.Kind(),
		Args: a, NextFire: e.Next(time.Now(), loc),
	})
}

// Migrate leva o esquema do Enxame, no banco da aplicação, à última
// versão. Pode rodar a cada início: aplica só o que falta.
func Migrate(ctx context.Context, pool *pgxpool.Pool) error {
	return postgres.Migrate(ctx, pool)
}

// RunResult é o estado de um run de workflow.
type RunResult struct {
	State  string // running, completed, failed
	Output json.RawMessage
	Err    string
}

// WorkflowResult devolve o estado de um run.
func (c *Client) WorkflowResult(
	ctx context.Context,
	runID string,
) (RunResult, error) {
	run, _, err := c.store.LoadRun(ctx, runID)
	return RunResult{State: string(run.State), Output: run.Output,
		Err: run.Err}, err
}

// Endpoint é um destino de webhooks.
type Endpoint struct {
	URL        string
	EventTypes []string // vazio: todos
	SecretRef  string   // env:NOME — o segredo whsec_… fica no ambiente
	RateLimit  int      // entregas por segundo; zero: sem limite
}

// CreateEndpoint inscreve um endpoint no namespace do cliente.
func (c *Client) CreateEndpoint(
	ctx context.Context,
	e Endpoint,
) (string, error) {
	n := nucleo.Endpoint{Namespace: c.namespace, URL: e.URL,
		EventTypes: e.EventTypes, SecretRef: e.SecretRef,
		RateLimit: e.RateLimit}
	if err := nucleo.Validar(n); err != nil {
		return "", err
	}
	n, err := c.store.CreateEndpoint(ctx, n)
	return n.ID, err
}
