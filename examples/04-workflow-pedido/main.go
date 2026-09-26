// Command workflow-pedido: um workflow de pedido com compensação —
// reservar o estoque, cobrar, esperar, emitir a nota; se a cobrança for
// recusada, liberar a reserva.
//
//	make up
//	export ENXAME_DB_DSN=postgres://…   # o de make up
//	go run ./examples/04-workflow-pedido
package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/go-sob-pressao/enxame/examples/internal/exemplo"
	"github.com/go-sob-pressao/enxame/pkg/enxame"
	"github.com/go-sob-pressao/enxame/pkg/job"
	"github.com/go-sob-pressao/enxame/pkg/workflow"
)

// Pedido é a entrada do workflow.
type Pedido struct {
	ID     string `json:"id"`
	Cartao string `json:"cartao"`
	Valor  int    `json:"valor"`
}

// livro:inicio exemplo-04

// ProcessarPedido é Go comum. Cada Step executa uma vez por run; se o
// processo morrer entre a cobrança e a nota, o replay devolve o recibo
// gravado e não cobra de novo.
func ProcessarPedido(
	c *workflow.Context,
	in json.RawMessage,
) (any, error) {
	var p Pedido
	if err := json.Unmarshal(in, &p); err != nil {
		return nil, err
	}
	reserva, err := workflow.Step(c, "reservar", reservar(p))
	if err != nil {
		return nil, err
	}
	recibo, err := workflow.Step(c, "cobrar", cobrar(p))
	if err != nil { // recusa definitiva: compensa a reserva
		_, errLib := workflow.Step(c, "liberar-reserva",
			liberar(reserva))
		return nil, errors.Join(err, errLib)
	}
	if err := workflow.Sleep(c, "prazo-de-arrependimento",
		2*time.Second); err != nil {
		return nil, err
	}
	return workflow.Step(c, "emitir-nota", emitirNota(p, recibo))
}

// livro:fim exemplo-04

func reservar(p Pedido) func(context.Context) (string, error) {
	return func(context.Context) (string, error) {
		fmt.Printf("%s: estoque reservado\n", p.ID)
		return "reserva-" + p.ID, nil
	}
}

func cobrar(p Pedido) func(context.Context) (string, error) {
	return func(ctx context.Context) (string, error) {
		if p.Cartao == "recusado" {
			fmt.Printf("%s: cartão recusado\n", p.ID)
			return "", workflow.Permanent(errors.New("cartão recusado"))
		}
		fmt.Printf("%s: cobrado, chave %s\n", p.ID,
			job.IdempotencyKey(ctx))
		return "recibo-" + p.ID, nil
	}
}

func liberar(reserva string) func(context.Context) (bool, error) {
	return func(context.Context) (bool, error) {
		fmt.Printf("%s liberada\n", reserva)
		return true, nil
	}
}

func emitirNota(p Pedido, recibo string) func(context.Context) (string,
	error) {
	return func(context.Context) (string, error) {
		fmt.Printf("%s: nota emitida para %s\n", p.ID, recibo)
		return "nota-" + p.ID, nil
	}
}

func main() {
	if err := rodar(context.Background()); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func rodar(ctx context.Context) error {
	db, err := exemplo.Banco(ctx, "enxame_exemplo04")
	if err != nil {
		return err
	}
	defer db.Close()
	client := enxame.New(db, "exemplo-04")
	var runs []string
	for _, p := range []Pedido{
		{ID: "P-1", Cartao: "ok", Valor: 100},
		{ID: "P-2", Cartao: "recusado", Valor: 100},
	} {
		id, err := client.StartWorkflow(ctx, "pedido", p.ID, p)
		if err != nil {
			return err
		}
		runs = append(runs, id)
	}
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	go exemplo.Esperar(ctx, db, cancel)
	w := client.NewWorker(enxame.WorkerConfig{})
	w.Workflow("pedido", ProcessarPedido)
	if err := w.Run(ctx); err != nil {
		return err
	}
	ctx = context.WithoutCancel(ctx) // o worker terminou; a leitura, não
	for _, id := range runs {
		r, err := client.WorkflowResult(ctx, id)
		if err != nil {
			return err
		}
		resultado := string(r.Output)
		if r.State != "completed" {
			resultado = r.Err
		}
		fmt.Printf("run %s…: %s — %s\n", id[:13], r.State, resultado)
	}
	return nil
}
