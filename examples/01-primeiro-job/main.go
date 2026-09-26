// Command primeiro-job: um handler, uma fila, um worker no mesmo
// processo — o modo biblioteca, só com pkg/enxame.
//
//	make up
//	export ENXAME_DB_DSN=postgres://…   # o de make up
//	go run ./examples/01-primeiro-job
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/go-sob-pressao/enxame/examples/internal/exemplo"
	"github.com/go-sob-pressao/enxame/pkg/enxame"
	"github.com/go-sob-pressao/enxame/pkg/job"
)

// EnviarEmail são os argumentos do job.
type EnviarEmail struct {
	Para    string `json:"para"`
	Assunto string `json:"assunto"`
}

// Kind escolhe o handler.
func (EnviarEmail) Kind() string { return "email.enviar" }

func main() {
	if err := rodar(context.Background()); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func rodar(ctx context.Context) error {
	db, err := exemplo.Banco(ctx, "enxame_exemplo01")
	if err != nil {
		return err
	}
	defer db.Close()
	client := enxame.New(db, "exemplo-01")

	para := []string{"ana@exemplo.com", "bia@exemplo.com",
		"caio@exemplo.com"}
	for _, p := range para {
		if _, err := client.Insert(ctx,
			EnviarEmail{Para: p, Assunto: "Bem-vindo"}); err != nil {
			return err
		}
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	go exemplo.Esperar(ctx, db, cancel) // até a fila esvaziar
	w := client.NewWorker(enxame.WorkerConfig{})
	w.Handle("email.enviar", func(ctx context.Context,
		j enxame.Job) error {
		var e EnviarEmail
		if err := json.Unmarshal(j.Args, &e); err != nil {
			return enxame.Permanent(err)
		}
		fmt.Printf("enviando para %s (chave %s…)\n", e.Para,
			job.IdempotencyKey(ctx)[:24])
		return nil
	})
	return w.Run(ctx)
}
