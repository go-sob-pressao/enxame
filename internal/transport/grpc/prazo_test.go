package grpc_test

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"

	"github.com/go-sob-pressao/enxame/internal/core/id"
	"github.com/go-sob-pressao/enxame/internal/core/job"
	tgrpc "github.com/go-sob-pressao/enxame/internal/transport/grpc"
	enxamev1 "github.com/go-sob-pressao/enxame/internal/transport/grpc/gen/enxame/v1"
)

// lento é um motor cujo batimento demora um segundo — ou menos, se o
// prazo da chamada acabar antes. Ele anota o prazo que recebeu.
type lento struct{ prazo chan time.Duration }

func (m *lento) Heartbeat(ctx context.Context, _ id.JobID, _ time.Time,
	_ int) error {
	d, ok := ctx.Deadline()
	if !ok {
		m.prazo <- -1
	} else {
		m.prazo <- time.Until(d)
	}
	select {
	case <-time.After(time.Second):
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (*lento) Claim(context.Context, string, time.Time,
	string) (job.Job, bool, error) {
	return job.Job{}, false, nil
}

func (*lento) Decide(context.Context, id.JobID,
	func(job.Job) ([]job.Event, error)) (job.Job, error) {
	return job.Job{}, nil
}

func cliente(t *testing.T, m tgrpc.Motor, padrao time.Duration) enxamev1.WorkerServiceClient {
	t.Helper()
	lis := bufconn.Listen(1 << 20)
	srv := grpc.NewServer(tgrpc.Servidor("t",
		slog.New(slog.DiscardHandler))...)
	enxamev1.RegisterWorkerServiceServer(srv,
		&tgrpc.Server{Motor: m, Poll: time.Second, Now: time.Now})
	go func() { _ = srv.Serve(lis) }()
	t.Cleanup(srv.Stop)
	opts := append(tgrpc.Cliente("t", padrao),
		grpc.WithContextDialer(func(ctx context.Context,
			_ string) (net.Conn, error) {
			return lis.DialContext(ctx)
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()))
	conn, err := grpc.NewClient("passthrough:///bufnet", opts...)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	return enxamev1.NewWorkerServiceClient(conn)
}

// livro:inicio prazo-teste

// O prazo do contexto do worker atravessa a rede: o handler do motor o
// recebe no próprio contexto, e para quando ele acaba. Uma chamada sem
// prazo recebe o padrão do interceptor.
func TestPrazoAtravessaARede(t *testing.T) {
	m := &lento{prazo: make(chan time.Duration, 2)}
	c := cliente(t, m, 300*time.Millisecond)
	req := &enxamev1.HeartbeatRequest{
		JobId: "0192a3b4-0000-7000-8000-000000000001", Attempt: 1}

	ctx, cancel := context.WithTimeout(t.Context(), 100*time.Millisecond)
	defer cancel()
	inicio := time.Now()
	_, err := c.Heartbeat(ctx, req)
	t.Logf("com prazo de 100 ms: motor viu %v; voltou em %v; %v",
		(<-m.prazo).Round(time.Millisecond),
		time.Since(inicio).Round(time.Millisecond), err)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("esperava DeadlineExceeded: %v", err)
	}

	inicio = time.Now()
	_, err = c.Heartbeat(t.Context(), req) // sem prazo nenhum
	t.Logf("sem prazo: motor viu %v; voltou em %v",
		(<-m.prazo).Round(time.Millisecond),
		time.Since(inicio).Round(time.Millisecond))
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("o padrão de 300 ms não valeu: %v", err)
	}
}

// livro:fim prazo-teste
