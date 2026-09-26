package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math/rand/v2"
	"net"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"

	"github.com/go-sob-pressao/enxame/internal/cluster/coordinator"
	"github.com/go-sob-pressao/enxame/internal/cluster/membership"
	"github.com/go-sob-pressao/enxame/internal/core/policy"
	"github.com/go-sob-pressao/enxame/internal/core/schedule"
	"github.com/go-sob-pressao/enxame/internal/delivery"
	"github.com/go-sob-pressao/enxame/internal/store/postgres"
	tgrpc "github.com/go-sob-pressao/enxame/internal/transport/grpc"
	enxamev1 "github.com/go-sob-pressao/enxame/internal/transport/grpc/gen/enxame/v1"
	api "github.com/go-sob-pressao/enxame/internal/transport/http"
	"github.com/go-sob-pressao/enxame/internal/worker"
	"github.com/go-sob-pressao/enxame/internal/worker/runner"
	"github.com/go-sob-pressao/enxame/pkg/webhook"
)

type config struct {
	dsn, http, grpc string
	tokens          map[string]string
	tokenWorker     string
	aviso, prazo    time.Duration
	resgate         time.Duration
	taxa, rajada    float64
	emCurso, naFila int
	no              string
}

// livro:inicio servir

// servir é o modo servidor: a API pública, o gRPC dos workers remotos e
// o motor — promover, resgatar, disparar agendamentos —, sob o mesmo
// errgroup. Quando ctx termina (SIGTERM), cada peça desliga em ordem, e
// servir só volta quando todas voltaram.
func servir(
	ctx context.Context,
	c config,
	lisHTTP, lisGRPC net.Listener,
	log *slog.Logger,
) error {
	db, err := pgxpool.New(ctx, c.dsn)
	if err != nil {
		return err
	}
	defer db.Close()
	if err := postgres.Migrate(ctx, db); err != nil {
		return err
	}
	s := postgres.New(db)
	g, ctx := errgroup.WithContext(ctx)

	m := membership.Novo(membership.Config{No: nomeDoNo(c.no),
		Endereco: lisHTTP.Addr().String(), DB: db, Log: log})
	g.Go(func() error { return m.Run(ctx) })

	a := api.NovaAPI(db, c.tokens, log)
	a.Cluster = m
	a.Taxa, a.Rajada, a.MaxEmCurso = c.taxa, c.rajada, c.emCurso
	a.Fila = api.Fila{Max: c.naFila, Validade: time.Second}
	g.Go(func() error {
		return a.Servir(ctx, lisHTTP,
			api.Desligamento{Aviso: c.aviso, Prazo: c.prazo})
	})

	// Cada chamada traz o próprio contexto, do stream.
	//nolint:contextcheck
	srv := grpc.NewServer(tgrpc.Servidor(c.tokenWorker, log)...)
	workers := &tgrpc.Server{Motor: s, Poll: 200 * time.Millisecond,
		Now: time.Now}
	enxamev1.RegisterWorkerServiceServer(srv, workers)
	g.Go(func() error { return srv.Serve(lisGRPC) })
	g.Go(func() error {
		<-ctx.Done()
		workers.Encerrar() // os streams de busca terminam
		pararGRPC(srv, c.prazo)
		return nil
	})

	g.Go(func() error { return motor(ctx, s, c.resgate) })
	g.Go(func() error { return entregar(ctx, s, log) })
	log.InfoContext(ctx, "enxamed no ar",
		slog.String("http", lisHTTP.Addr().String()),
		slog.String("grpc", lisGRPC.Addr().String()))
	if err := g.Wait(); !errors.Is(err, context.Canceled) {
		return err
	}
	return nil
}

// livro:fim servir

// nomeDoNo usa o nome dado, ou máquina-pid.
func nomeDoNo(nome string) coordinator.NodeID {
	if nome != "" {
		return coordinator.NodeID(nome)
	}
	h, _ := os.Hostname()
	return coordinator.NodeID(fmt.Sprintf("%s-%d", h, os.Getpid()))
}

// pararGRPC espera os streams terminarem; se não terminarem no prazo,
// fecha tudo. Um stream de long-poll só termina quando o worker desiste
// — por isso o prazo.
func pararGRPC(srv *grpc.Server, prazo time.Duration) {
	feito := make(chan struct{})
	go func() {
		srv.GracefulStop()
		close(feito)
	}()
	select {
	case <-feito:
	case <-time.After(prazo):
		srv.Stop()
	}
}

// motor é o papel de motor no modo servidor: o que o worker embutido
// faz no modo biblioteca.
func motor(
	ctx context.Context,
	s *postgres.Store,
	resgate time.Duration,
) error {
	t := time.NewTicker(time.Second)
	defer t.Stop()
	for {
		agora := time.Now()
		if _, err := s.Promote(ctx, agora); err != nil {
			return err
		}
		if _, err := s.Rescue(ctx, agora,
			agora.Add(-resgate)); err != nil {
			return err
		}
		if _, _, err := s.FireDue(ctx, agora,
			schedule.Disparo); err != nil {
			return err
		}
		select {
		case <-t.C:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

// entregar roda o pool da entrega de webhooks: uma fila própria, com
// workers próprios — um bulkhead: a entrega lenta a um cliente não
// ocupa os workers dos jobs.
func entregar(
	ctx context.Context,
	s *postgres.Store,
	log *slog.Logger,
) error {
	d := delivery.Novo(s)
	retry := policy.Retry{Base: 30 * time.Second, Max: time.Hour}
	p := &worker.Pool{Queue: postgres.NewFila(ctx, s),
		QueueName: webhook.FanoutQueue, Concurrency: 16,
		Handlers: map[string]runner.Handler{
			webhook.FanoutKind:  d.Fanout,
			webhook.DeliverKind: d.Entregar},
		PollTimeout: 5 * time.Second, AttemptTimeout: 20 * time.Second,
		Backoff: func(a int) time.Duration {
			return retry.Delay(a, rand.Float64)
		},
		ReportEvery: time.Minute, Worker: "enxamed-entrega",
		Now: time.Now, Log: log}
	return p.Run(ctx)
}
