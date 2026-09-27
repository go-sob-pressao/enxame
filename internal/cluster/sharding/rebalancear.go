package sharding

import (
	"context"
	"log/slog"
	"maps"
	"slices"
	"time"

	"github.com/go-sob-pressao/enxame/internal/cluster/coordinator"
	"github.com/go-sob-pressao/enxame/internal/engine/partition"
	"github.com/go-sob-pressao/enxame/internal/simulation/clock"
)

// Posse é o que o rebalanceador usa do lease das partições. Em
// produção, partition.Lease, sobre o banco; na simulação, o modelo do
// banco simulado.
type Posse interface {
	AdquirirVarias(ctx context.Context, ps []int) (map[int]int64, error)
	RenovarVarias(ctx context.Context,
		posses map[int]int64) (map[int]int64, error)
	Soltar(ctx context.Context, p int, token int64) error
}

// EmCurso conta, para cada partição de ps, as tentativas em execução.
type EmCurso func(ctx context.Context, ps []int) (map[int]int, error)

// Rebalanceador faz as posses deste nó acompanharem o mapa do
// coordenador: larga, depois de drenar, o que o mapa tirou dele, e
// adquire o que o mapa deu. Não cria goroutine nenhuma: cada Passo faz
// o que cabe naquele instante, e quem chama Passo decide quando — um
// ticker em produção, o escalonador na simulação.
type Rebalanceador struct {
	No        coordinator.NodeID
	Coord     coordinator.Coordinator
	Lease     Posse
	Posses    *partition.Posses
	EmCurso   EmCurso
	Relogio   clock.Clock   // nil: o relógio do sistema
	Intervalo time.Duration // renovação e leitura do mapa (1 s)
	Drenagem  time.Duration // teto para esperar as tentativas (30 s)
	Log       *slog.Logger

	epoca    uint64
	proxima  time.Time        // próxima renovação e leitura do mapa
	meus     []int            // as partições que o mapa dá a este nó
	drenando map[int]drenagem // partições saindo, à espera de soltar
}

type drenagem struct {
	token  int64
	limite time.Time
}

// Run chama Passo cinco vezes por intervalo, até ctx terminar.
func (r *Rebalanceador) Run(ctx context.Context) error {
	t := time.NewTicker(r.Intervalo / 5)
	defer t.Stop()
	for {
		r.Passo(ctx)
		select {
		case <-ctx.Done():
			return nil
		case <-t.C:
		}
	}
}

// livro:inicio rebalancear

// Passo renova as posses e lê o mapa, uma vez por intervalo; e, a cada
// chamada, solta as partições drenadas e tenta adquirir as que faltam.
func (r *Rebalanceador) Passo(ctx context.Context) {
	agora := r.agora()
	if !agora.Before(r.proxima) {
		r.proxima = agora.Add(r.Intervalo)
		r.renovar(ctx)
		a, err := r.Coord.Assignment(ctx)
		if err == nil && a.Epoch > r.epoca &&
			len(a.Owners) == coordinator.NumPartitions {
			r.aplicar(a, agora)
			r.epoca = a.Epoch
		}
	}
	r.soltarDrenadas(ctx, agora)
	r.adquirir(ctx)
}

// aplicar leva as posses ao mapa a, em três fases. Drenar: as partições
// que saem deixam de receber trabalho novo, e cada uma espera as
// tentativas em curso terminarem. Liberar: só então o lease é
// devolvido, e o próximo dono não precisa esperar o vencimento.
// Adquirir: as partições que entram são tomadas assim que o dono
// anterior as libera.
func (r *Rebalanceador) aplicar(
	a coordinator.Assignment,
	agora time.Time,
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
				r.drenar(p, token, agora)
			}
		}
	}
	r.meus = slices.Sorted(maps.Keys(meus))
}

// livro:fim rebalancear

// livro:inicio adquirir

// adquirir tenta, a cada passo, toda partição que o mapa dá a este nó
// e que ele ainda não tem. Não há prazo para desistir: uma partição do
// mapa sem posse é uma partição parada, e só este nó vai tomá-la.
func (r *Rebalanceador) adquirir(ctx context.Context) {
	todas := r.Posses.Todas()
	var faltam []int
	for _, p := range r.meus {
		if _, tem := todas[p]; !tem {
			faltam = append(faltam, p)
		}
	}
	if len(faltam) == 0 {
		return
	}
	ganhas, err := r.Lease.AdquirirVarias(ctx, faltam)
	if err != nil {
		return
	}
	for p, token := range ganhas {
		r.Posses.Pegar(p, token)
	}
}

// livro:fim adquirir

func (r *Rebalanceador) drenar(p int, token int64, agora time.Time) {
	if r.drenando == nil {
		r.drenando = map[int]drenagem{}
	}
	r.drenando[p] = drenagem{token: token,
		limite: agora.Add(r.Drenagem)}
}

// soltarDrenadas devolve o lease das partições em drenagem que não têm
// mais tentativas em curso — ou cuja Drenagem venceu.
func (r *Rebalanceador) soltarDrenadas(
	ctx context.Context,
	agora time.Time,
) {
	if len(r.drenando) == 0 {
		return
	}
	ps := slices.Sorted(maps.Keys(r.drenando))
	emCurso, err := r.EmCurso(ctx, ps)
	if err != nil {
		return
	}
	for _, p := range ps {
		d := r.drenando[p]
		if emCurso[p] > 0 && agora.Before(d.limite) {
			continue
		}
		_ = r.Lease.Soltar(context.WithoutCancel(ctx), p, d.token)
		r.Posses.Largar(p)
		delete(r.drenando, p)
	}
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
			delete(r.drenando, p)
		}
	}
}

func (r *Rebalanceador) agora() time.Time {
	if r.Relogio == nil {
		return time.Now()
	}
	return r.Relogio.Now()
}
