package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"time"
	"uuid"

	"github.com/go-sob-pressao/enxame/internal/core/id"
	"github.com/go-sob-pressao/enxame/internal/core/job"
	"github.com/go-sob-pressao/enxame/internal/queue"
	"github.com/go-sob-pressao/enxame/internal/worker/runner"
)

// livro:inicio demo-m0

// demonstrar executa o M0: n jobs "eco" numa fila em memória, um por
// vez. O terceiro job falha na primeira tentativa, para mostrar o
// retry.
func demonstrar(n int, saida, erros io.Writer) int {
	q := queue.NewMemory()
	for i := range n {
		s := job.Spec{
			ID:    id.JobID(uuid.NewV7()),
			Queue: "padrao", Kind: "eco",
			Args: fmt.Appendf(nil, `{"n":%d}`, i+1),
		}
		if _, err := q.Insert(s, time.Now()); err != nil {
			fmt.Fprintln(erros, "enfileirar:", err)
			return 1
		}
	}

	falhou := false
	r := &runner.Sequential{
		Queue:     q,
		QueueName: "padrao",
		Worker:    "enxamed-m0",
		Now:       time.Now,
		Handlers: map[string]runner.Handler{
			"eco": func(_ context.Context, j job.Job) error {
				fmt.Fprintf(
					saida,
					"tentativa %d do job %s: %s\n",
					j.Attempt,
					j.ID,
					j.Args,
				)
				if !falhou && string(j.Args) == `{"n":3}` {
					falhou = true
					return errors.New("falha simulada")
				}
				return nil
			},
		},
	}
	executados, err := r.Drain(context.Background())
	if err != nil {
		fmt.Fprintln(erros, "executar:", err)
		return 1
	}
	fmt.Fprintf(saida, "%d jobs, %d execuções\n", n, executados)
	return 0
}

// livro:fim demo-m0
