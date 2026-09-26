//go:build integration

package integration_test

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"sync/atomic"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"

	"github.com/go-sob-pressao/enxame/internal/core/job"
	"github.com/go-sob-pressao/enxame/internal/store/postgres"
	tgrpc "github.com/go-sob-pressao/enxame/internal/transport/grpc"
	enxamev1 "github.com/go-sob-pressao/enxame/internal/transport/grpc/gen/enxame/v1"
	"github.com/go-sob-pressao/enxame/internal/worker"
	"github.com/go-sob-pressao/enxame/internal/worker/poller"
	"github.com/go-sob-pressao/enxame/internal/worker/runner"
	"github.com/go-sob-pressao/enxame/test/testutil"
)

const token = "segredo-de-teste"

// motor sobe o servidor gRPC sobre o Postgres, numa porta livre.
func motor(t *testing.T, s *postgres.Store) string {
	t.Helper()
	var lc net.ListenConfig
	lis, err := lc.Listen(t.Context(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	srv := grpc.NewServer(tgrpc.Servidor(token,
		slog.New(slog.DiscardHandler))...)
	enxamev1.RegisterWorkerServiceServer(srv, &tgrpc.Server{
		Motor: s, Poll: 50 * time.Millisecond, Now: time.Now})
	go func() { _ = srv.Serve(lis) }()
	t.Cleanup(srv.Stop)
	return lis.Addr().String()
}

func conexao(t *testing.T, addr, tok string) *grpc.ClientConn {
	t.Helper()
	opts := append(tgrpc.Cliente(tok, 5*time.Second),
		grpc.WithTransportCredentials(insecure.NewCredentials()))
	conn, err := grpc.NewClient(addr, opts...)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	return conn
}

func poolRemoto(
	ctx context.Context,
	t *testing.T,
	conn *grpc.ClientConn,
	nome string,
	h runner.Handler,
) *worker.Pool {
	t.Helper()
	r, err := poller.Conectar(ctx, conn, nome, []string{"q"}, 2)
	if err != nil {
		t.Fatal(err)
	}
	return &worker.Pool{Queue: r, QueueName: "q", Concurrency: 4,
		Handlers:    map[string]runner.Handler{"eco": h},
		PollTimeout: time.Second, AttemptTimeout: 10 * time.Second,
		RetryDelay: time.Millisecond, ReportEvery: time.Hour,
		Heartbeat: r.Heartbeat, HeartbeatEvery: 200 * time.Millisecond,
		Worker: nome, Now: time.Now, Log: slog.New(slog.DiscardHandler)}
}

// livro:inicio remoto-teste

// Um worker remoto, noutra ponta de uma conexão gRPC, executa 100 jobs
// do motor; o pool é o mesmo da Parte I, com a fila remota.
func TestWorkerRemoto(t *testing.T) {
	s := postgres.New(testutil.Postgres(t))
	ids := enfileirar(t, s, 100)
	conn := conexao(t, motor(t, s), token)
	var n atomic.Int64
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	p := poolRemoto(ctx, t, conn, "remoto-1",
		func(context.Context, job.Job) error { n.Add(1); return nil })
	go func() { _ = p.Run(ctx) }()
	esperarConcluidos(t, s, ids)
	if n.Load() != 100 {
		t.Fatalf("%d execuções", n.Load())
	}
}

// O erro de domínio atravessa a rede como código, e volta a ser o erro
// de domínio: errors.Is funciona do lado do worker.
func TestErroAtravessaARede(t *testing.T) {
	s := postgres.New(testutil.Postgres(t))
	ids := enfileirar(t, s, 1)
	addr := motor(t, s)
	c := enxamev1.NewWorkerServiceClient(conexao(t, addr, token))
	_, err := c.Heartbeat(t.Context(), &enxamev1.HeartbeatRequest{
		JobId: ids[0].String(), Attempt: 1}) // o job nem começou
	t.Logf("erro recebido: %v", err)
	if !errors.Is(err, job.ErrInvalidTransition) {
		t.Fatalf("errors.Is falhou: %v", err)
	}
	intruso := enxamev1.NewWorkerServiceClient(conexao(t, addr, "x"))
	_, err = intruso.Heartbeat(t.Context(),
		&enxamev1.HeartbeatRequest{JobId: ids[0].String()})
	if status.Code(err) != codes.Unauthenticated {
		t.Fatalf("sem token: %v", err)
	}
}

// livro:fim remoto-teste

// O worker remoto some no meio do job: a conexão cai, os batimentos
// param, e o resgate do motor devolve o job; outro worker o termina.
func TestWorkerRemotoMorre(t *testing.T) {
	s := postgres.New(testutil.Postgres(t))
	ids := enfileirar(t, s, 1)
	addr := motor(t, s)
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	go func() { // o resgate do motor: prazo de 700 ms
		for ctx.Err() == nil {
			agora := time.Now()
			_, _ = s.Rescue(ctx, agora, agora.Add(-700*time.Millisecond))
			<-time.After(100 * time.Millisecond)
		}
	}()
	comecou := make(chan struct{})
	conn1 := conexao(t, addr, token)
	ctx1, morre := context.WithCancel(ctx)
	p1 := poolRemoto(ctx1, t, conn1, "remoto-1",
		func(ctx context.Context, _ job.Job) error {
			close(comecou)
			<-ctx.Done() // trabalha até morrer
			return ctx.Err()
		})
	go func() { _ = p1.Run(ctx1) }()
	<-comecou
	morre()
	_ = conn1.Close() // o processo morreu: nenhuma mensagem sai mais

	p2 := poolRemoto(ctx, t, conexao(t, addr, token), "remoto-2",
		func(context.Context, job.Job) error { return nil })
	go func() { _ = p2.Run(ctx) }()
	esperarConcluidos(t, s, ids)
	j, _, _ := s.Get(t.Context(), ids[0])
	if j.Attempt != 2 || j.AttemptedBy != "remoto-2" {
		t.Fatalf("tentativa %d por %s", j.Attempt, j.AttemptedBy)
	}
}
