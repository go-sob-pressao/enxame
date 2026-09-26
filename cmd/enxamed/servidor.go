package main

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"

	"github.com/go-sob-pressao/enxame/internal/core/schedule"
	"github.com/go-sob-pressao/enxame/internal/store/postgres"
	tgrpc "github.com/go-sob-pressao/enxame/internal/transport/grpc"
	enxamev1 "github.com/go-sob-pressao/enxame/internal/transport/grpc/gen/enxame/v1"
	api "github.com/go-sob-pressao/enxame/internal/transport/http"
)

type config struct {
	dsn, http, grpc string
	tokens          map[string]string
	tokenWorker     string
	aviso, prazo    time.Duration
	resgate         time.Duration
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

	a := api.NovaAPI(db, c.tokens, log)
	g.Go(func() error {
		return a.Servir(ctx, lisHTTP,
			api.Desligamento{Aviso: c.aviso, Prazo: c.prazo})
	})

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
	log.InfoContext(ctx, "enxamed no ar",
		slog.String("http", lisHTTP.Addr().String()),
		slog.String("grpc", lisGRPC.Addr().String()))
	if err := g.Wait(); !errors.Is(err, context.Canceled) {
		return err
	}
	return nil
}

// livro:fim servir

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
