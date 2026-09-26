// Command cron-e-longo-prazo: um agendamento a cada minuto, disparado
// pelo worker embutido, com a chave da janela no job.
//
//	make up
//	export ENXAME_DB_DSN=postgres://…   # o de make up
//	go run ./examples/05-cron-e-longo-prazo
//
// Espera o próximo minuto cheio — até 60 segundos.
package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/go-sob-pressao/enxame/examples/internal/exemplo"
	"github.com/go-sob-pressao/enxame/pkg/enxame"
	"github.com/go-sob-pressao/enxame/pkg/job"
)

// FecharCaixa são os argumentos do job agendado.
type FecharCaixa struct {
	Loja string `json:"loja"`
}

// Kind escolhe o handler.
func (FecharCaixa) Kind() string { return "caixa.fechar" }

func main() {
	if err := rodar(context.Background()); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func rodar(ctx context.Context) error {
	db, err := exemplo.Banco(ctx, "enxame_exemplo05")
	if err != nil {
		return err
	}
	defer db.Close()
	client := enxame.New(db, "exemplo-05")
	if err := client.Schedule(ctx, enxame.Schedule{
		ID: "fechamento", Cron: "* * * * *",
		Timezone: "America/Sao_Paulo",
		Args:     FecharCaixa{Loja: "centro"},
	}); err != nil {
		return err
	}
	fmt.Printf("agendado; agora são %s\n",
		time.Now().Format("15:04:05"))
	ctx, cancel := context.WithTimeout(ctx, 75*time.Second)
	defer cancel()
	w := client.NewWorker(enxame.WorkerConfig{})
	w.Handle("caixa.fechar", func(ctx context.Context,
		_ enxame.Job) error {
		fmt.Printf("%s: fechando o caixa (chave %s)\n",
			time.Now().Format("15:04:05"), job.IdempotencyKey(ctx))
		cancel() // um disparo basta para o exemplo
		return nil
	})
	if err := w.Run(ctx); err != nil && ctx.Err() == nil {
		return err
	}
	return nil
}
