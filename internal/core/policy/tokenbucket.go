package policy

import "time"

// livro:inicio tokenbucket

// TokenBucket é o balde de fichas: enche à Taxa por segundo, até
// Rajada fichas; cada chamada gasta uma. A Taxa é o ritmo sustentado,
// e a Rajada, quanto se pode gastar de uma vez depois de um tempo
// parado. É puro: o instante vem de fora, e o balde não dorme — diz
// quanto esperar, e quem chama decide o que fazer com a espera.
type TokenBucket struct {
	Taxa   float64 // fichas por segundo
	Rajada float64 // capacidade do balde

	fichas float64
	ultimo time.Time
}

// Tomar gasta uma ficha, se houver. Sem ficha, devolve false e quanto
// esperar até a próxima.
func (b *TokenBucket) Tomar(agora time.Time) (bool, time.Duration) {
	if b.ultimo.IsZero() {
		b.fichas, b.ultimo = b.Rajada, agora
	}
	if agora.After(b.ultimo) {
		b.fichas = min(b.Rajada,
			b.fichas+agora.Sub(b.ultimo).Seconds()*b.Taxa)
		b.ultimo = agora
	}
	if b.fichas >= 1 {
		b.fichas--
		return true, 0
	}
	falta := (1 - b.fichas) / b.Taxa
	return false, time.Duration(falta * float64(time.Second))
}

// livro:fim tokenbucket
