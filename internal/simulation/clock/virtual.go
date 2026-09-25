package clock

import (
	"cmp"
	"slices"
	"sync"
	"time"
)

// livro:inicio virtual

// Virtual é um relógio que só anda quando alguém chama Advance. Os
// timers disparam na ordem dos prazos, com o relógio parado em cada
// prazo no instante do disparo: nada depende da velocidade da máquina.
type Virtual struct {
	mu     sync.Mutex
	mudou  sync.Cond // sinaliza timers novos, para BlockUntil
	agora  time.Time
	seq    int
	timers []*virtualTimer
}

// NewVirtual cria um relógio parado em inicio.
func NewVirtual(inicio time.Time) *Virtual {
	v := &Virtual{agora: inicio}
	v.mudou.L = &v.mu
	return v
}

// Now devolve a hora virtual.
func (v *Virtual) Now() time.Time {
	v.mu.Lock()
	defer v.mu.Unlock()
	return v.agora
}

// NewTimer cria um timer que dispara quando o relógio chegar a
// agora+d. Com d <= 0, dispara no próximo Advance, mesmo de zero.
func (v *Virtual) NewTimer(d time.Duration) Timer {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.seq++
	t := &virtualTimer{
		v:     v,
		c:     make(chan time.Time, 1),
		prazo: v.agora.Add(d),
		seq:   v.seq,
	}
	v.timers = append(v.timers, t)
	v.mudou.Broadcast()
	return t
}

// BlockUntil espera até existirem n timers pendentes. É o sinal que o
// teste precisa para saber que o código já pediu para ser acordado —
// e, portanto, que o próximo Advance vai acordá-lo.
func (v *Virtual) BlockUntil(n int) {
	v.mu.Lock()
	defer v.mu.Unlock()
	for len(v.timers) < n {
		v.mudou.Wait()
	}
}

// Advance anda d, disparando em ordem os timers vencidos no caminho.
func (v *Virtual) Advance(d time.Duration) {
	v.mu.Lock()
	defer v.mu.Unlock()
	alvo := v.agora.Add(d)
	slices.SortFunc(v.timers, func(a, b *virtualTimer) int {
		return cmp.Or(
			a.prazo.Compare(b.prazo),
			cmp.Compare(a.seq, b.seq),
		)
	})
	for len(v.timers) > 0 && !v.timers[0].prazo.After(alvo) {
		t := v.timers[0]
		v.timers = v.timers[1:]
		// um prazo que já passou não faz o relógio voltar
		if t.prazo.After(v.agora) {
			v.agora = t.prazo
		}
		t.c <- v.agora // buffer de 1: nunca bloqueia
	}
	v.agora = alvo
}

// livro:fim virtual

type virtualTimer struct {
	v     *Virtual
	c     chan time.Time
	prazo time.Time
	seq   int
}

func (t *virtualTimer) C() <-chan time.Time { return t.c }

// Stop cancela o timer; devolve false se ele já tinha disparado.
func (t *virtualTimer) Stop() bool {
	t.v.mu.Lock()
	defer t.v.mu.Unlock()
	i := slices.Index(t.v.timers, t)
	if i < 0 {
		return false
	}
	t.v.timers = slices.Delete(t.v.timers, i, i+1)
	return true
}
