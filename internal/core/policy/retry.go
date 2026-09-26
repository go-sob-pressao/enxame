package policy

import "time"

// livro:inicio retry

// Retry é a política de espera entre tentativas: backoff exponencial a
// partir de Base, limitado por Max, com jitter total — a espera é
// sorteada entre zero e o teto exponencial.
type Retry struct {
	Base time.Duration
	Max  time.Duration
}

// Delay devolve a espera antes da tentativa seguinte à tentativa
// attempt (1, 2, 3…). aleatorio devolve um número em [0, 1); vem de
// fora para que a política continue pura e a simulação, reproduzível.
func (r Retry) Delay(
	attempt int,
	aleatorio func() float64,
) time.Duration {
	teto := r.Base
	for i := 1; i < attempt && teto < r.Max; i++ {
		teto *= 2
	}
	teto = min(teto, r.Max)
	return time.Duration(aleatorio() * float64(teto))
}

// livro:fim retry

// Teto devolve a espera sem jitter, para comparação.
func (r Retry) Teto(attempt int) time.Duration {
	return r.Delay(attempt, func() float64 { return 1 })
}
