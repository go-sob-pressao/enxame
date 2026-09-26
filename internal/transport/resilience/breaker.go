package resilience

import (
	"errors"
	"sync"
	"time"
)

// ErrAberto indica que o circuito está aberto: a chamada nem é feita.
var ErrAberto = errors.New("circuito aberto")

// Estado é o estado de um circuito.
type Estado int

// Os três estados de um circuit breaker.
const (
	Fechado    Estado = iota // tudo passa; conta as falhas seguidas
	Aberto                   // nada passa, até a pausa acabar
	MeioAberto               // uma chamada de teste passa
)

func (e Estado) String() string {
	return [...]string{"fechado", "aberto", "meio-aberto"}[e]
}

// livro:inicio breaker

// Breakers guarda um circuito por chave — no Enxame, por endpoint de
// webhook. Limiar falhas seguidas abrem o circuito; aberto, ele recusa
// as chamadas por Pausa; depois, deixa passar uma só, de teste: se ela
// der certo, fecha; se falhar, abre de novo, com outra pausa.
type Breakers struct {
	Limiar int
	Pausa  time.Duration

	mu        sync.Mutex
	circuitos map[string]*circuito
}

type circuito struct {
	estado   Estado
	falhas   int
	abriuEm  time.Time
	testando bool // a chamada de teste do meio-aberto está no ar
}

// Permitir diz se a chamada para chave pode ser feita agora.
func (b *Breakers) Permitir(chave string, agora time.Time) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	c := b.circuito(chave)
	if c.estado == Aberto && agora.Sub(c.abriuEm) >= b.Pausa {
		c.estado = MeioAberto
	}
	switch {
	case c.estado == Aberto:
		return ErrAberto
	case c.estado == MeioAberto && c.testando:
		return ErrAberto // o teste é um só
	case c.estado == MeioAberto:
		c.testando = true
	}
	return nil
}

// Registrar conta o resultado de uma chamada permitida.
func (b *Breakers) Registrar(chave string, agora time.Time, ok bool) {
	b.mu.Lock()
	defer b.mu.Unlock()
	c := b.circuito(chave)
	c.testando = false
	switch {
	case ok:
		c.estado, c.falhas = Fechado, 0
	case c.estado == MeioAberto:
		c.estado, c.abriuEm = Aberto, agora
	default:
		c.falhas++
		if c.falhas >= b.Limiar {
			c.estado, c.abriuEm = Aberto, agora
		}
	}
}

// livro:fim breaker

// Estado devolve o estado do circuito de chave.
func (b *Breakers) Estado(chave string, agora time.Time) Estado {
	b.mu.Lock()
	defer b.mu.Unlock()
	c := b.circuito(chave)
	if c.estado == Aberto && agora.Sub(c.abriuEm) >= b.Pausa {
		return MeioAberto
	}
	return c.estado
}

func (b *Breakers) circuito(chave string) *circuito {
	if b.circuitos == nil {
		b.circuitos = map[string]*circuito{}
	}
	c, ok := b.circuitos[chave]
	if !ok {
		c = &circuito{}
		b.circuitos[chave] = c
	}
	return c
}
