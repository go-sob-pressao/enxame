// Package liderperdido é a Missão #5: o líder perdido.
package liderperdido

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/go-sob-pressao/enxame/internal/cluster/coordinator"
	"github.com/go-sob-pressao/enxame/internal/cluster/membership"
	"github.com/go-sob-pressao/enxame/internal/cluster/pgcoord"
	"github.com/go-sob-pressao/enxame/internal/cluster/sharding"
	"github.com/go-sob-pressao/enxame/internal/engine/partition"
	"github.com/go-sob-pressao/enxame/internal/store/postgres"
)

// No é um nó do cluster da missão: membership, coordenador,
// rebalanceador e promotor, com tempos curtos para o teste.
type No struct {
	Nome   coordinator.NodeID
	Coord  *pgcoord.Coord
	Posses *partition.Posses
	parar  func()
}

// Subir põe um nó no ar, sobre db.
func Subir(db *pgxpool.Pool, nome coordinator.NodeID) *No {
	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup
	log := slog.New(slog.DiscardHandler)
	m := membership.Novo(membership.Config{No: nome, DB: db,
		Intervalo: 100 * time.Millisecond,
		Detector: func() membership.Detector {
			p := membership.NovoPhi(100 * time.Millisecond)
			p.Folga = 300 * time.Millisecond
			return p
		}})
	wg.Go(func() { _ = m.Run(ctx) })
	c := pgcoord.New(ctx, pgcoord.Config{ID: nome, Pool: db,
		Membros: m, Intervalo: 100 * time.Millisecond,
		Lease: 500 * time.Millisecond, Log: log})
	posses := &partition.Posses{}
	r := &sharding.Rebalanceador{No: nome, Coord: c, DB: db,
		Lease: partition.Lease{DB: db, No: string(nome),
			Duracao: time.Second},
		Posses: posses, Intervalo: 200 * time.Millisecond,
		Drenagem: time.Second, Espera: 2 * time.Second, Log: log}
	wg.Go(func() { _ = r.Run(ctx) })
	p := &Promotor{Store: postgres.New(db).RelogioDoBanco(),
		Posses: posses}
	wg.Go(func() {
		for ctx.Err() == nil {
			_ = p.Passar(ctx, time.Now())
			<-time.After(200 * time.Millisecond)
		}
	})
	return &No{Nome: nome, Coord: c, Posses: posses,
		parar: func() { _ = c.Close(); cancel(); wg.Wait() }}
}

// Parar derruba o nó sem devolver nada: as posses vencem sozinhas.
func (n *No) Parar() { n.parar() }
