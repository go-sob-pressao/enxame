// Command remoto: o Experimento 18.1 — o motor num processo, workers
// remotos em outros, conversando por gRPC.
//
//	go run ./examples/cap18/remoto motor          # terminal 1
//	go run ./examples/cap18/remoto worker w1      # terminal 2
//	kill -9 <pid do w1>, no meio do trabalho
//	go run ./examples/cap18/remoto worker w2      # terminal 3
//
// O motor recria o banco enxame_remoto, enfileira 12 jobs de um
// segundo e resgata tentativas sem sinal de vida há mais de 3 segundos.
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"os"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/go-sob-pressao/enxame/examples/internal/exemplo"
	"github.com/go-sob-pressao/enxame/internal/core/job"
	"github.com/go-sob-pressao/enxame/internal/store/postgres"
	tgrpc "github.com/go-sob-pressao/enxame/internal/transport/grpc"
	enxamev1 "github.com/go-sob-pressao/enxame/internal/transport/grpc/gen/enxame/v1"
	"github.com/go-sob-pressao/enxame/internal/worker"
	"github.com/go-sob-pressao/enxame/internal/worker/poller"
	"github.com/go-sob-pressao/enxame/internal/worker/runner"
	"github.com/go-sob-pressao/enxame/pkg/enxame"
)

const (
	endereco = "127.0.0.1:7233"
	token    = "enxame-local"
)

// Longo são os argumentos do job de exemplo.
type Longo struct {
	N int `json:"n"`
}

// Kind escolhe o handler.
func (Longo) Kind() string { return "longo" }

func main() {
	var err error
	switch {
	case len(os.Args) > 1 && os.Args[1] == "motor":
		err = motor(context.Background())
	case len(os.Args) > 2 && os.Args[1] == "worker":
		err = trabalhar(context.Background(), os.Args[2])
	default:
		err = errors.New("uso: remoto motor | remoto worker NOME")
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func motor(ctx context.Context) error {
	db, err := exemplo.Banco(ctx, "enxame_remoto")
	if err != nil {
		return err
	}
	defer db.Close()
	c := enxame.New(db, "remoto")
	for i := range 12 {
		if _, err := c.Insert(ctx, Longo{N: i},
			enxame.Queue("q")); err != nil {
			return err
		}
	}
	s := postgres.New(db)
	var lc net.ListenConfig
	lis, err := lc.Listen(ctx, "tcp", endereco)
	if err != nil {
		return err
	}
	srv := grpc.NewServer(tgrpc.Servidor(token,
		slog.New(slog.DiscardHandler))...)
	enxamev1.RegisterWorkerServiceServer(srv, &tgrpc.Server{
		Motor: s, Poll: 100 * time.Millisecond, Now: time.Now})
	go func() { _ = srv.Serve(lis) }()
	defer srv.Stop()
	fmt.Println("motor: 12 jobs na fila q, ouvindo em", endereco)
	for {
		agora := time.Now()
		n, err := s.Rescue(ctx, agora, agora.Add(-3*time.Second))
		if err != nil {
			return err
		}
		if n > 0 {
			fmt.Printf("motor: %d tentativa(s) sem sinal de vida "+
				"resgatada(s)\n", n)
		}
		var pendentes int
		if err := db.QueryRow(ctx, `SELECT count(*) FROM job
			WHERE state <> 'completed'`).Scan(&pendentes); err != nil {
			return err
		}
		if pendentes == 0 {
			fmt.Println("motor: todos os jobs concluídos")
			return nil
		}
		<-time.After(500 * time.Millisecond)
	}
}

func trabalhar(ctx context.Context, nome string) error {
	opts := append(tgrpc.Cliente(token, 5*time.Second),
		grpc.WithTransportCredentials(insecure.NewCredentials()))
	conn, err := grpc.NewClient(endereco, opts...)
	if err != nil {
		return err
	}
	defer func() { _ = conn.Close() }()
	r, err := poller.Conectar(ctx, conn, nome, []string{"q"}, 1)
	if err != nil {
		return err
	}
	longo := func(ctx context.Context, j job.Job) error {
		fmt.Printf("%s: job %s…, tentativa %d\n", nome,
			j.ID.String()[24:], j.Attempt)
		select {
		case <-time.After(time.Second):
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	p := &worker.Pool{Queue: r, QueueName: "q", Concurrency: 2,
		Handlers:    map[string]runner.Handler{"longo": longo},
		PollTimeout: time.Second, AttemptTimeout: time.Minute,
		RetryDelay: time.Second, ReportEvery: time.Hour,
		Heartbeat: r.Heartbeat, HeartbeatEvery: 500 * time.Millisecond,
		Worker: nome, Now: time.Now, Log: slog.New(slog.DiscardHandler)}
	return p.Run(ctx)
}
