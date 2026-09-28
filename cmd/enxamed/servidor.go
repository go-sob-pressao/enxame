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
	"github.com/go-sob-pressao/enxame/internal/cluster/pgcoord"
	"github.com/go-sob-pressao/enxame/internal/cluster/routing"
	"github.com/go-sob-pressao/enxame/internal/cluster/sharding"
	"github.com/go-sob-pressao/enxame/internal/core/policy"
	"github.com/go-sob-pressao/enxame/internal/core/schedule"
	"github.com/go-sob-pressao/enxame/internal/delivery"
	"github.com/go-sob-pressao/enxame/internal/engine/partition"
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
	leaseMotor      time.Duration
	relogio         func() time.Time // o relógio de parede do nó
	diag            string           // endereço do pprof; vazio: não
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
	if c.relogio == nil {
		c.relogio = time.Now
	}
	db, err := pgxpool.New(ctx, c.dsn)
	if err != nil {
		return err
	}
	defer db.Close()
	if err := postgres.Migrate(ctx, db); err != nil {
		return err
	}
	// Um relógio só para todos os nós: o do banco (Caps. 22 e 24).
	s := postgres.New(db).RelogioDoBanco()
	g, ctx := errgroup.WithContext(ctx)
	no := nomeDoNo(c.no)

	m := membership.Novo(membership.Config{No: no,
		Endereco: anunciado(lisHTTP.Addr()), DB: db, Log: log})
	g.Go(func() error { return m.Run(ctx) })

	// Quem manda e de quem é cada partição (Caps. 23 a 26): o pgcoord
	// escreve o mapa; o rebalanceador faz as posses segui-lo; o Store
	// "do dono" só reserva, promove e resgata nas partições do nó, com
	// o token de cada uma conferido.
	coord := pgcoord.New(ctx, pgcoord.Config{ID: no, Pool: db,
		Membros: m, Intervalo: 500 * time.Millisecond,
		Lease: 3 * time.Second, Log: log})
	defer func() { _ = coord.Close() }()
	posses := &partition.Posses{}
	reb := &sharding.Rebalanceador{No: no, Coord: coord,
		EmCurso: sharding.EmCursoNoBanco(db),
		Lease: partition.Lease{DB: db, No: string(no),
			Duracao: c.leaseMotor},
		Posses: posses, Intervalo: time.Second,
		Drenagem: 30 * time.Second, Log: log}
	g.Go(func() error { return reb.Run(ctx) })
	sd := s.ComDono(postgres.Dono{Tokens: posses.Tokens,
		Perdeu: posses.Largar})

	d := delivery.Novo(sd)
	a := api.NovaAPI(db, c.tokens, log)
	a.Cluster, a.Entrega = m, d
	a.Rota = routing.Rota{No: no, Coord: coord,
		Tenho: func(p int) bool {
			_, ok := posses.Tokens()[p]
			return ok
		},
		Endereco: func(n coordinator.NodeID) string {
			for _, s := range m.Visao() {
				if s.No == n {
					return s.Endereco
				}
			}
			return ""
		}}
	a.Taxa, a.Rajada, a.MaxEmCurso = c.taxa, c.rajada, c.emCurso
	a.Fila = api.Fila{Max: c.naFila, Validade: time.Second}
	g.Go(func() error {
		return a.Servir(ctx, lisHTTP,
			api.Desligamento{Aviso: c.aviso, Prazo: c.prazo})
	})

	// Cada chamada traz o próprio contexto, do stream.
	//nolint:contextcheck
	srv := grpc.NewServer(tgrpc.Servidor(c.tokenWorker, log)...)
	workers := &tgrpc.Server{Motor: sd, Poll: 200 * time.Millisecond,
		Now: c.relogio}
	enxamev1.RegisterWorkerServiceServer(srv, workers)
	g.Go(func() error { return srv.Serve(lisGRPC) })
	g.Go(func() error {
		<-ctx.Done()
		workers.Encerrar() // os streams de busca terminam
		pararGRPC(srv, c.prazo)
		return nil
	})

	g.Go(func() error {
		return motor(ctx, sd, coord, no, c.resgate, c.relogio, log)
	})
	g.Go(func() error {
		return worker.Supervisionar(ctx, log, "entrega",
			func(ctx context.Context) error {
				return entregar(ctx, sd, d, c.relogio, log)
			})
	})
	if c.diag != "" {
		var lc net.ListenConfig
		lisDiag, err := lc.Listen(ctx, "tcp", c.diag)
		if err != nil {
			return err
		}
		g.Go(func() error { return diagnosticar(ctx, lisDiag, log) })
	}
	log.InfoContext(ctx, "enxamed no ar",
		slog.String("http", lisHTTP.Addr().String()),
		slog.String("grpc", lisGRPC.Addr().String()))
	if err := g.Wait(); !errors.Is(err, context.Canceled) {
		return err
	}
	return nil
}

// livro:fim servir

// anunciado é o endereço que os outros nós e os clientes usam: sem
// host, o da máquina local.
func anunciado(a net.Addr) string {
	host, porta, err := net.SplitHostPort(a.String())
	if err != nil || host == "" || host == "::" || host == "0.0.0.0" {
		host = "127.0.0.1"
	}
	return net.JoinHostPort(host, porta)
}

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
// livro:inicio motor

// motor promove e resgata nas partições deste nó — o Store do dono
// confere os tokens de todas numa transação —, e dispara os
// agendamentos se este nó for o líder: os disparos não têm partição, e
// a chave de cada janela absorve o que um líder antigo repetir.
func motor(
	ctx context.Context,
	s *postgres.Store,
	coord coordinator.Coordinator,
	no coordinator.NodeID,
	resgate time.Duration,
	relogio func() time.Time,
	log *slog.Logger,
) error {
	t := time.NewTicker(time.Second)
	defer t.Stop()
	for {
		// Um erro do banco — um deadlock, uma conexão caída — vale para
		// este ciclo; o próximo tenta de novo. Derrubar o nó por ele
		// seria trocar um erro passageiro por um rebalanceamento.
		agora := relogio()
		if _, err := s.Promote(ctx, agora); err != nil {
			avisar(ctx, log, "promover", err)
		}
		if _, err := s.Rescue(ctx, agora,
			agora.Add(-resgate)); err != nil {
			avisar(ctx, log, "resgatar", err)
		}
		if l, _ := coord.Leader(ctx); l == no {
			if _, _, err := s.FireDue(ctx, agora,
				schedule.Disparo); err != nil {
				avisar(ctx, log, "disparar", err)
			}
		}
		select {
		case <-t.C:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

// livro:fim motor

func avisar(
	ctx context.Context,
	log *slog.Logger,
	o string,
	err error,
) {
	if ctx.Err() == nil {
		log.WarnContext(ctx, "motor: "+o, slog.Any("erro", err))
	}
}

// entregar roda o pool da entrega de webhooks: uma fila própria, com
// workers próprios — um bulkhead: a entrega lenta a um cliente não
// ocupa os workers dos jobs.
func entregar(
	ctx context.Context,
	s *postgres.Store,
	d *delivery.Entregador,
	relogio func() time.Time,
	log *slog.Logger,
) error {
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
		Now: relogio, Log: log}
	return p.Run(ctx)
}
