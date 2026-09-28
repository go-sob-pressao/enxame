// Command realidade4 é o Teste de Realidade #4 (Capítulo 32): uma
// atualização de versão do Enxame em Kubernetes com 5.000 jobs e 500
// workflows em andamento.
//
//	realidade4 worker -dsn … -grpc enxame-0.enxame-nos:7233,…
//	realidade4 executar -dsn … -api http://localhost:30080 \
//	    -atualizar 'kubectl set image statefulset/enxame …'
//
// O worker roda jobs de meio segundo e um workflow de dois passos com
// uma espera de 60 s no meio; cada efeito conta as próprias execuções.
// O executar enfileira pela API, dispara a atualização, amostra o banco
// e a API a cada 500 ms e, no fim, confere que nada se perdeu.
package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "uso: realidade4 worker|executar")
		os.Exit(2)
	}
	fs := flag.NewFlagSet(os.Args[1], flag.ExitOnError)
	dsn := fs.String("dsn", os.Getenv("ENXAME_DB_DSN"), "PostgreSQL")
	addr := fs.String("grpc", "enxame-0.enxame-nos:7233",
		"gRPC de cada enxamed, separados por vírgula")
	nome := fs.String("nome", os.Getenv("HOSTNAME"), "nome do worker")
	var e execucao
	fs.StringVar(&e.api, "api", "http://localhost:30080", "API")
	fs.IntVar(&e.jobs, "jobs", 5000, "jobs")
	fs.IntVar(&e.workflows, "workflows", 500, "workflows")
	fs.DurationVar(&e.duracao, "duracao", 100*time.Second,
		"tempo para enfileirar os jobs, em ritmo constante")
	fs.DurationVar(&e.aos, "aos", 20*time.Second,
		"quando disparar a atualização")
	fs.StringVar(&e.atualizar, "atualizar", "",
		"comando de shell que dispara a atualização")
	fs.StringVar(&e.esperar, "esperar", "",
		"comando de shell que espera a atualização terminar")
	fs.StringVar(&e.csv, "csv", "realidade4.csv", "linha do tempo")
	_ = fs.Parse(os.Args[2:])
	ctx, parar := signal.NotifyContext(context.Background(),
		os.Interrupt, syscall.SIGTERM)
	defer parar()
	db, err := pgxpool.New(ctx, *dsn)
	if err == nil {
		defer db.Close()
		switch os.Args[1] {
		case "worker":
			log := slog.New(slog.NewJSONHandler(os.Stderr, nil))
			err = trabalhar(ctx, db, *addr, *nome, log)
		case "executar":
			e.db = db
			err = e.executar(ctx)
		default:
			err = fmt.Errorf("comando desconhecido: %s", os.Args[1])
		}
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// Trabalho é o argumento dos jobs; Pedido, a entrada dos workflows.
type (
	Trabalho struct {
		N int `json:"n"`
	}
	Pedido struct {
		N int `json:"n"`
	}
)
