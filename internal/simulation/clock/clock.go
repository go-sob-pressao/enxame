package clock

import "time"

// livro:inicio clock

// Clock é o relógio injetado: quem precisa de tempo recebe um Clock em
// vez de chamar time.Now e time.NewTimer. Em produção, Real; nos
// testes e na simulação, Virtual.
type Clock interface {
	Now() time.Time
	NewTimer(d time.Duration) Timer
}

// Timer é o que o Enxame usa de um time.Timer.
type Timer interface {
	C() <-chan time.Time
	Stop() bool
}

// Real é o relógio do sistema.
type Real struct{}

// Now devolve a hora do sistema.
func (Real) Now() time.Time { return time.Now() }

// NewTimer cria um time.Timer de verdade.
func (Real) NewTimer(d time.Duration) Timer {
	return realTimer{time.NewTimer(d)}
}

type realTimer struct{ t *time.Timer }

func (r realTimer) C() <-chan time.Time { return r.t.C }
func (r realTimer) Stop() bool          { return r.t.Stop() }

// livro:fim clock
