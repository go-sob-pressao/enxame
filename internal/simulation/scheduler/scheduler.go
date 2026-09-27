package scheduler

import (
	"container/heap"
	"math/rand/v2"
	"time"
)

// livro:inicio scheduler

// Scheduler executa eventos em tempo virtual, um de cada vez, na ordem
// (instante, ordem de agendamento). Não há goroutine nenhuma: a mesma
// seed produz a mesma sequência de sorteios e, portanto, a mesma
// execução, byte a byte.
type Scheduler struct {
	agora time.Duration
	fila  eventos
	seq   uint64
	rng   *rand.Rand
}

// New cria um escalonador cuja única fonte de acaso vem de seed.
func New(seed uint64) *Scheduler {
	return &Scheduler{rng: rand.New(rand.NewPCG(seed, ^seed))}
}

// Now é o instante virtual corrente.
func (s *Scheduler) Now() time.Duration { return s.agora }

// Rand é a única fonte de acaso da simulação: todo sorteio passa por
// ela.
func (s *Scheduler) Rand() *rand.Rand { return s.rng }

// After agenda fn para daqui a d, em tempo virtual.
func (s *Scheduler) After(d time.Duration, fn func()) {
	s.seq++
	heap.Push(&s.fila, evento{quando: s.agora + d, seq: s.seq, fn: fn})
}

// Run executa os eventos até o instante limite, avançando o relógio
// virtual de evento em evento — não há espera real.
func (s *Scheduler) Run(limite time.Duration) {
	for s.fila.Len() > 0 && s.fila[0].quando <= limite {
		e := heap.Pop(&s.fila).(evento)
		s.agora = e.quando
		e.fn()
	}
	s.agora = limite
}

// livro:fim scheduler

type evento struct {
	quando time.Duration
	seq    uint64
	fn     func()
}

type eventos []evento

func (e eventos) Len() int { return len(e) }
func (e eventos) Less(i, j int) bool {
	if e[i].quando != e[j].quando {
		return e[i].quando < e[j].quando
	}
	return e[i].seq < e[j].seq
}
func (e eventos) Swap(i, j int) { e[i], e[j] = e[j], e[i] }
func (e *eventos) Push(x any)   { *e = append(*e, x.(evento)) }
func (e *eventos) Pop() any {
	v := (*e)[len(*e)-1]
	*e = (*e)[:len(*e)-1]
	return v
}
