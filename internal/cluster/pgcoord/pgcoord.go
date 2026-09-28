package pgcoord

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"slices"
	"sync"
	"sync/atomic"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/go-sob-pressao/enxame/internal/cluster/coordinator"
)

// Config configura um coordenador.
type Config struct {
	ID        coordinator.NodeID
	Pool      *pgxpool.Pool
	Membros   coordinator.Membership // quem está vivo (Cap. 23)
	Intervalo time.Duration          // renovação do lease e releitura
	Lease     time.Duration          // quanto a liderança dura
	Log       *slog.Logger
}

// Coord implementa coordinator.Coordinator sobre o PostgreSQL: lease
// de liderança numa linha de controle, com o termo como fencing token,
// e um mapa por época, gravado só pelo líder. Quem está vivo vem do
// membership do Capítulo 23. O relógio que vale é o now() do banco.
type Coord struct {
	cfg       Config
	Consultas atomic.Int64 // comandos enviados ao banco pelo laço

	mu    sync.Mutex
	termo int64 // o termo em que este nó é líder; 0 se não for
	lider coordinator.NodeID
	mapa  coordinator.Assignment

	parar context.CancelFunc
	fim   chan struct{}
}

// New inicia o coordenador; ele roda até Close. O ciclo de vida é o do
// coordenador, não o de ctx: só os valores de ctx são herdados.
func New(ctx context.Context, cfg Config) *Coord {
	ctx, parar := context.WithCancel(context.WithoutCancel(ctx))
	c := &Coord{cfg: cfg, parar: parar, fim: make(chan struct{})}
	go c.laco(ctx)
	return c
}

func (c *Coord) laco(ctx context.Context) {
	defer close(c.fim)
	t := time.NewTicker(c.cfg.Intervalo)
	defer t.Stop()
	for {
		if err := c.ciclo(ctx); err != nil && ctx.Err() == nil {
			c.cfg.Log.WarnContext(
				ctx,
				"pgcoord: ciclo falhou",
				slog.String(
					"no",
					string(c.cfg.ID),
				),
				slog.Any("erro", err),
			)
		}
		select {
		case <-ctx.Done():
			return
		case <-t.C:
		}
	}
}

// ciclo: lease, redistribuição (só no líder), releitura.
func (c *Coord) ciclo(ctx context.Context) error {
	termo, err := c.adquirirOuRenovar(ctx)
	if err != nil {
		return err
	}
	if termo > 0 {
		if err := c.redistribuir(ctx, termo); err != nil {
			return err
		}
	}
	return c.reler(ctx, termo)
}

// livro:inicio pgcoord-lideranca

// adquirirOuRenovar: o UPDATE só acontece se o lease venceu ou já é
// deste nó; o termo só cresce quando a liderança muda de dono.
func (c *Coord) adquirirOuRenovar(ctx context.Context) (int64, error) {
	c.Consultas.Add(1)
	var termo int64
	err := c.cfg.Pool.QueryRow(ctx, `
		UPDATE coord_leader
		   SET term = CASE WHEN node = $1 THEN term ELSE term + 1 END,
		       node = $1,
		       expires_at = now() + $2::interval
		 WHERE id = 1 AND (node = $1 OR expires_at < now())
		RETURNING term`, c.cfg.ID, c.cfg.Lease).Scan(&termo)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, nil
	}
	return termo, err
}

// livro:fim pgcoord-lideranca

// livro:inicio pgcoord-mapa

// redistribuir grava um mapa novo se os vivos não são os donos do mapa
// corrente. O FOR SHARE na linha do líder é o fencing: um líder antigo,
// que acordou depois de perder o lease, lê um termo diferente do seu e
// desiste — e, se a troca de dono acontecer depois da leitura, o UPDATE
// dela espera esta transação terminar (ADR-005).
func (c *Coord) redistribuir(ctx context.Context, termo int64) error {
	c.Consultas.Add(1)
	return pgx.BeginFunc(ctx, c.cfg.Pool, func(tx pgx.Tx) error {
		var atual int64
		const fencing = `
			SELECT term FROM coord_leader WHERE id = 1 FOR SHARE`
		if err := tx.QueryRow(ctx, fencing).Scan(&atual); err != nil {
			return err
		}
		if atual != termo {
			return nil // perdemos a liderança no caminho
		}
		vivos, err := c.cfg.Membros.Members(ctx)
		if errors.Is(err, coordinator.ErrSemVisao) {
			return nil // o próximo ciclo decide, já com a visão
		}
		if err != nil {
			return err
		}
		anterior, err := ultimoMapa(ctx, tx)
		if err != nil {
			return err
		}
		if slices.Equal(donos(anterior), vivos) {
			return nil
		}
		novo := coordinator.Distribute(vivos, anterior)
		owners, _ := json.Marshal(novo.Owners)
		_, err = tx.Exec(ctx, `
			INSERT INTO coord_assignment (epoch, term, owners)
			VALUES ($1, $2, $3)`,
			novo.Epoch, termo, owners)
		return err
	})
}

// livro:fim pgcoord-mapa

func ultimoMapa(ctx context.Context, q interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}) (coordinator.Assignment, error) {
	var a coordinator.Assignment
	var owners []byte
	err := q.QueryRow(ctx, `
		SELECT epoch, owners FROM coord_assignment
		 ORDER BY epoch DESC LIMIT 1`).
		Scan(&a.Epoch, &owners)
	if errors.Is(err, pgx.ErrNoRows) {
		return a, nil
	}
	if err != nil {
		return a, err
	}
	return a, json.Unmarshal(owners, &a.Owners)
}

// reler atualiza o que Leader e Assignment devolvem.
func (c *Coord) reler(ctx context.Context, termo int64) error {
	c.Consultas.Add(2)
	var lider *string
	if err := c.cfg.Pool.QueryRow(ctx, `
		SELECT CASE WHEN expires_at > now() THEN node END
		  FROM coord_leader WHERE id = 1`).Scan(&lider); err != nil {
		return err
	}
	mapa, err := ultimoMapa(ctx, c.cfg.Pool)
	if err != nil {
		return err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.termo, c.mapa, c.lider = termo, mapa, ""
	if lider != nil {
		c.lider = coordinator.NodeID(*lider)
	}
	return nil
}

func donos(a coordinator.Assignment) []coordinator.NodeID {
	return slices.Compact(slices.Sorted(slices.Values(a.Owners)))
}

// Leader devolve o líder com lease válido na última leitura.
func (c *Coord) Leader(context.Context) (coordinator.NodeID, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.lider, nil
}

// Members devolve a visão do membership deste nó.
func (c *Coord) Members(
	ctx context.Context,
) ([]coordinator.NodeID, error) {
	return c.cfg.Membros.Members(ctx)
}

// Assignment devolve o mapa corrente.
func (c *Coord) Assignment(
	context.Context,
) (coordinator.Assignment, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	a := c.mapa
	a.Owners = slices.Clone(a.Owners)
	return a, nil
}

// Close para o coordenador. O lease não é liberado: vence sozinho — é o
// que acontece quando um nó morre, e é o caso que importa testar.
func (c *Coord) Close() error {
	c.parar()
	<-c.fim
	return nil
}
