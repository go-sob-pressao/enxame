package partition

import (
	"maps"
	"sync"
)

// Posses são as partições deste nó: as ativas, onde ele reserva,
// promove e resgata; e as em drenagem, que ele está entregando e em que
// não começa nada novo. Cada uma com o token da posse.
type Posses struct {
	mu       sync.Mutex
	ativas   map[int]int64
	drenando map[int]int64
}

// Tokens devolve uma cópia das posses ativas.
func (p *Posses) Tokens() map[int]int64 {
	p.mu.Lock()
	defer p.mu.Unlock()
	return maps.Clone(p.ativas)
}

// Todas devolve as ativas e as em drenagem: todas se renovam.
func (p *Posses) Todas() map[int]int64 {
	p.mu.Lock()
	defer p.mu.Unlock()
	t := maps.Clone(p.ativas)
	if t == nil {
		t = map[int]int64{}
	}
	maps.Copy(t, p.drenando)
	return t
}

// Pegar registra uma posse nova, ativa.
func (p *Posses) Pegar(particao int, token int64) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.ativas == nil {
		p.ativas = map[int]int64{}
	}
	p.ativas[particao] = token
}

// Drenar tira a partição das ativas: nada novo começa nela.
func (p *Posses) Drenar(particao int) (int64, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	token, ok := p.ativas[particao]
	if !ok {
		return 0, false
	}
	delete(p.ativas, particao)
	if p.drenando == nil {
		p.drenando = map[int]int64{}
	}
	p.drenando[particao] = token
	return token, true
}

// Largar esquece a partição, ativa ou em drenagem.
func (p *Posses) Largar(particao int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	delete(p.ativas, particao)
	delete(p.drenando, particao)
}
