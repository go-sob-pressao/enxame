//go:build integration

package integration_test

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/go-sob-pressao/enxame/internal/store/postgres"
	"github.com/go-sob-pressao/enxame/pkg/enxame"
	"github.com/go-sob-pressao/enxame/pkg/webhook"
	"github.com/go-sob-pressao/enxame/pkg/workflow"
	"github.com/go-sob-pressao/enxame/test/testutil"
)

type cobrarPedido struct {
	PedidoID int `json:"pedido_id"`
}

func (cobrarPedido) Kind() string { return "cobrar-pedido" }

func contar(t *testing.T, db *pgxpool.Pool, sql string) int {
	t.Helper()
	var n int
	if err := db.QueryRow(t.Context(), sql).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

// O pedido, o job de cobrança, a mensagem de webhook e o run: tudo na
// transação da aplicação. ROLLBACK leva tudo; COMMIT grava tudo.
func TestTudoOuNadaNaTransacaoDaAplicacao(t *testing.T) {
	db := testutil.Postgres(t)
	ctx := t.Context()
	if _, err := db.Exec(ctx, `CREATE TABLE pedido
		(id INT PRIMARY KEY, total INT NOT NULL)`); err != nil {
		t.Fatal(err)
	}
	c := enxame.New(db, "loja")
	gravar := func(tx pgx.Tx, n int) error {
		if _, err := tx.Exec(ctx,
			`INSERT INTO pedido VALUES ($1, 100)`, n); err != nil {
			return err
		}
		if _, err := c.InsertTx(ctx, tx, cobrarPedido{n},
			enxame.UniqueKey(fmt.Sprintf("cobranca:%d", n))); err != nil {
			return err
		}
		if _, err := c.PublishTx(ctx, tx, webhook.Message{
			EventType: "pedido.criado", Payload: map[string]int{"id": n},
		}); err != nil {
			return err
		}
		_, err := c.StartWorkflowTx(ctx, tx, "entrega",
			fmt.Sprintf("pedido-%d", n), nil)
		return err
	}
	falha := errors.New("estoque indisponível")
	err := pgx.BeginFunc(ctx, db, func(tx pgx.Tx) error {
		if err := gravar(tx, 1); err != nil {
			return err
		}
		return falha // a aplicação desiste depois de enfileirar
	})
	if !errors.Is(err, falha) {
		t.Fatal(err)
	}
	if err := pgx.BeginFunc(ctx, db, func(tx pgx.Tx) error {
		return gravar(tx, 2)
	}); err != nil {
		t.Fatal(err)
	}
	for sql, esperado := range map[string]int{
		`SELECT count(*) FROM pedido`:                            1,
		`SELECT count(*) FROM job WHERE kind = 'cobrar-pedido'`:  1,
		`SELECT count(*) FROM webhook_message`:                   1,
		`SELECT count(*) FROM job WHERE kind = 'webhook.fanout'`: 1,
		`SELECT count(*) FROM workflow_run`:                      1,
		`SELECT count(*) FROM job
		   WHERE kind = 'workflow.avancar'`: 1,
	} {
		if n := contar(t, db, sql); n != esperado {
			t.Errorf("%s: %d; esperado %d", sql, n, esperado)
		}
	}
	// A chave única também vale dentro da transação da aplicação.
	err = pgx.BeginFunc(ctx, db, func(tx pgx.Tx) error {
		_, err := c.InsertTx(ctx, tx, cobrarPedido{2},
			enxame.UniqueKey("cobranca:2"))
		return err
	})
	if !errors.Is(err, enxame.ErrDuplicate) {
		t.Fatalf("segunda cobrança do pedido 2: %v", err)
	}
}

// livro:inicio falha-injetada

// Falha injetada no meio da transação de transição: o job da posição
// seguinte não pode ser gravado (a chave única já está ocupada). O
// passo, que foi gravado antes na mesma transação, também não fica.
func TestTransicaoNaoGravaMetade(t *testing.T) {
	db := testutil.Postgres(t)
	s := postgres.New(db)
	ctx := t.Context()
	runID, err := s.StartRun(ctx, workflow.Run{
		Namespace: "ns", WorkflowID: "w", Type: "f"})
	if err != nil {
		t.Fatal(err)
	}
	// Ocupa a chave do job da posição 2.
	c := enxame.New(db, "ns")
	if _, err := c.Insert(ctx, cobrarPedido{0}, enxame.UniqueKey(
		"workflow:"+runID+":2")); err != nil {
		t.Fatal(err)
	}
	err = s.AppendStep(ctx, runID,
		workflow.Record{Seq: 1, Name: "cobrar",
			Kind: workflow.KindCall, Output: []byte(`"R-1"`)},
		workflow.Continuation{Seq: 2, At: time.Now()})
	if !errors.Is(err, enxame.ErrDuplicate) {
		t.Fatalf("AppendStep: %v", err)
	}
	_, passos, err := s.LoadRun(ctx, runID)
	if err != nil || len(passos) != 0 {
		t.Fatalf("passos gravados: %v, %v", passos, err)
	}
}

// livro:fim falha-injetada
