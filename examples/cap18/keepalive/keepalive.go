// Package keepalive é o enigma do Capítulo 18: o keepalive agressivo
// do cliente que o servidor pune com GOAWAY.
package keepalive

import (
	"context"
	"log/slog"
	"net"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"

	"github.com/go-sob-pressao/enxame/internal/core/id"
	"github.com/go-sob-pressao/enxame/internal/core/job"
	tgrpc "github.com/go-sob-pressao/enxame/internal/transport/grpc"
	enxamev1 "github.com/go-sob-pressao/enxame/internal/transport/grpc/gen/enxame/v1"
)

// livro:inicio keepalive

// Cliente é o worker do incidente: para que o balanceador não derrube o
// stream ocioso, manda um ping a cada 10 segundos, mesmo sem chamada
// em andamento.
var Cliente = keepalive.ClientParameters{
	Time:                10 * time.Second,
	Timeout:             5 * time.Second,
	PermitWithoutStream: true,
}

// Padrao é o que o servidor aceita sem configuração nenhuma: no mínimo
// cinco minutos entre pings, e nenhum ping sem stream aberto.
var Padrao = keepalive.EnforcementPolicy{MinTime: 5 * time.Minute}

// Alinhada é a política do servidor combinada com o cliente.
var Alinhada = keepalive.EnforcementPolicy{
	MinTime:             5 * time.Second,
	PermitWithoutStream: true,
}

// livro:fim keepalive

// Servidor sobe um motor sem jobs, com a política dada.
func Servidor(
	lis net.Listener,
	politica keepalive.EnforcementPolicy,
) *grpc.Server {
	opts := append(tgrpc.Servidor("t", slog.New(slog.DiscardHandler)),
		grpc.KeepaliveEnforcementPolicy(politica))
	srv := grpc.NewServer(opts...)
	enxamev1.RegisterWorkerServiceServer(srv, &tgrpc.Server{
		Motor: vazio{}, Poll: time.Hour, Now: time.Now})
	go func() { _ = srv.Serve(lis) }()
	return srv
}

// Esperar abre o stream de busca, como um worker ocioso, e devolve o
// erro que o encerrar — ou nil, se ele durar até o fim de ctx.
func Esperar(ctx context.Context, addr string) error {
	opts := append(tgrpc.Cliente("t", time.Second),
		grpc.WithKeepaliveParams(Cliente),
		grpc.WithTransportCredentials(insecure.NewCredentials()))
	conn, err := grpc.NewClient(addr, opts...)
	if err != nil {
		return err
	}
	defer func() { _ = conn.Close() }()
	stream, err := enxamev1.NewWorkerServiceClient(conn).Fetch(ctx)
	if err != nil {
		return err
	}
	if err := stream.Send(&enxamev1.FetchRequest{Worker: "w",
		Queues: []string{"q"}, Credit: 1}); err != nil {
		return err
	}
	_, err = stream.Recv()
	if ctx.Err() != nil {
		return nil
	}
	return err
}

type vazio struct{}

func (vazio) Claim(context.Context, string, time.Time,
	string) (job.Job, bool, error) {
	return job.Job{}, false, nil
}

func (vazio) Heartbeat(
	context.Context, id.JobID, time.Time, int,
) error {
	return nil
}

func (vazio) Decide(context.Context, id.JobID,
	func(job.Job) ([]job.Event, error)) (job.Job, error) {
	return job.Job{}, nil
}
