package timer

import (
	"container/heap"
	"context"
	"time"

	"github.com/go-sob-pressao/enxame/internal/simulation/clock"
)

// livro:inicio pump

// Pump acorda na hora do próximo prazo e chama Promote, que torna
// disponíveis os jobs cuja hora chegou. Os prazos ficam num min-heap:
// o próximo está sempre no topo, e o pump dorme até ele — nem antes,
// varrendo a fila à toa, nem depois, atrasando o job.
type Pump struct {
	clock   clock.Clock
	promote func(at time.Time) (int, error)
	novos   chan time.Time
	prazos  minHeap
}

// NewPump cria um pump. promote recebe o instante do disparo.
func NewPump(
	c clock.Clock,
	promote func(at time.Time) (int, error),
) *Pump {
	return &Pump{
		clock:   c,
		promote: promote,
		novos:   make(chan time.Time),
	}
}

// Schedule avisa o pump de que há trabalho elegível em at.
func (p *Pump) Schedule(ctx context.Context, at time.Time) error {
	select {
	case p.novos <- at:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Run executa até o contexto terminar ou Promote falhar.
func (p *Pump) Run(ctx context.Context) error {
	for {
		var espera <-chan time.Time // nil: sem prazo, o caso some
		var t clock.Timer
		if p.prazos.Len() > 0 {
			t = p.clock.NewTimer(p.prazos[0].Sub(p.clock.Now()))
			espera = t.C()
		}
		select {
		case <-ctx.Done():
			parar(t)
			return ctx.Err()
		case at := <-p.novos:
			heap.Push(&p.prazos, at)
		case <-espera:
			agora := p.clock.Now()
			for p.prazos.Len() > 0 && !p.prazos[0].After(agora) {
				heap.Pop(&p.prazos)
			}
			if _, err := p.promote(agora); err != nil {
				return err
			}
		}
		parar(t)
	}
}

// livro:fim pump

func parar(t clock.Timer) {
	if t != nil {
		t.Stop()
	}
}

// minHeap guarda os prazos, o menor no índice 0.
type minHeap []time.Time

func (h minHeap) Len() int           { return len(h) }
func (h minHeap) Less(i, j int) bool { return h[i].Before(h[j]) }
func (h minHeap) Swap(i, j int)      { h[i], h[j] = h[j], h[i] }
func (h *minHeap) Push(x any)        { *h = append(*h, x.(time.Time)) }

func (h *minHeap) Pop() any {
	velho := *h
	n := len(velho)
	x := velho[n-1]
	*h = velho[:n-1]
	return x
}
