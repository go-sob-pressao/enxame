package sharding

import (
	"context"
	"log/slog"
	"slices"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/go-sob-pressao/enxame/internal/cluster/coordinator"
	"github.com/go-sob-pressao/enxame/internal/engine/partition"
)

// Rebalanceador faz as posses deste nó acompanharem o mapa do
// coordenador: larga, depois de drenar, o que o mapa tirou dele, e
// adquire o que o mapa deu.
type Rebalanceador struct {
	No        coordinator.NodeID
	Coord     coordinator.Coordinator
	Lease     partition.Lease
	Posses    *partition.Posses
	DB        *pgxpool.Pool
	Intervalo time.Duration // renovação e leitura do mapa (1 s)
	Drenagem  time.Duration // teto para esperar as tentativas (30 s)
	Espera    time.Duration // quanto insistir na aquisição (2 s)
	Log       *slog.Logger

	epoca uint64
}

// livro:inicio rebalancear

// Run renova as posses e segue o mapa, a cada intervalo, até ctx
// terminar.
func (r *Rebalanceador) Run(ctx context.Context) error {
	t := time.NewTicker(r.Intervalo)
	defer t.Stop()
	for {
		r.renovar(ctx)
		a, err := r.Coord.Assignment(ctx)
		if err == nil && a.Epoch > r.epoca &&
			len(a.Owners) == coordinator.NumPartitions {
			r.aplicar(ctx, a)
			r.epoca = a.Epoch
		}
		select {
		case <-ctx.Done():
			return nil
		case <-t.C:
		}
	}
}

// aplicar leva as posses ao mapa a, em três fases. Drenar: as partições
// que saem deixam de receber trabalho novo, e cada uma espera, numa
// goroutine própria, as tentativas em curso terminarem. Liberar: só
// então o lease é devolvido, e o próximo dono não precisa esperar o
// vencimento. Adquirir: as partições que entram são tomadas assim que o
// dono anterior as libera.
func (r *Rebalanceador) aplicar(
	ctx context.Context,
	a coordinator.Assignment,
) {
	meus := map[int]bool{}
	for p, dono := range a.Owners {
		if dono == r.No {
			meus[p] = true
		}
	}
	for p := range r.Posses.Tokens() {
		if !meus[p] {
			if token, ok := r.Posses.Drenar(p); ok {
				go r.drenarELiberar(ctx, p, token)
			}
		}
	}
	var faltam []int
	todas := r.Posses.Todas()
	for p := range meus {
		if _, tem := todas[p]; !tem {
			faltam = append(faltam, p)
		}
	}
	r.adquirir(ctx, faltam)
}

// adquirir insiste nas partições que faltam por até Espera: a drenagem
// do dono anterior costuma levar menos que isso.
func (r *Rebalanceador) adquirir(ctx context.Context, faltam []int) {
	limite := time.Now().Add(r.Espera)
	for len(faltam) > 0 && time.Now().Before(limite) &&
		ctx.Err() == nil {
		ganhas, err := r.Lease.AdquirirVarias(ctx, faltam)
		if err == nil {
			for p, token := range ganhas {
				r.Posses.Pegar(p, token)
			}
			faltam = slices.DeleteFunc(faltam, func(p int) bool {
				_, ok := ganhas[p]
				return ok
			})
		}
		if len(faltam) > 0 {
			<-time.After(r.Intervalo / 5)
		}
	}
	if len(faltam) > 0 {
		r.Log.WarnContext(ctx, "partições não adquiridas",
			slog.Int("quantas", len(faltam)))
	}
}

// livro:fim rebalancear

// drenarELiberar espera as tentativas em curso na partição terminarem
// — ou a Drenagem vencer — e devolve o lease.
func (r *Rebalanceador) drenarELiberar(
	ctx context.Context,
	p int,
	token int64,
) {
	limite := time.Now().Add(r.Drenagem)
	for time.Now().Before(limite) && ctx.Err() == nil {
		var emCurso int
		err := r.DB.QueryRow(ctx, `SELECT count(*) FROM job
			WHERE partition_id = $1 AND state = 'running'`, p).
			Scan(&emCurso)
		if err == nil && emCurso == 0 {
			break
		}
		<-time.After(r.Intervalo / 5)
	}
	_ = r.Lease.Soltar(context.WithoutCancel(ctx), p, token)
	r.Posses.Largar(p)
}

// renovar estende todas as posses; as que o banco não renovou foram
// perdidas — para outro nó, com um token maior — e saem.
func (r *Rebalanceador) renovar(ctx context.Context) {
	todas := r.Posses.Todas()
	if len(todas) == 0 {
		return
	}
	mantidas, err := r.Lease.RenovarVarias(ctx, todas)
	if err != nil {
		return // o banco não respondeu: o fencing cuida do resto
	}
	for p := range todas {
		if _, ok := mantidas[p]; !ok {
			r.Posses.Largar(p)
		}
	}
}
