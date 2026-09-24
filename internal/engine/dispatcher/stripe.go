package dispatcher

import (
	"hash/maphash"
	"sync"
)

// KeyLock serializa operações sobre a mesma chave (job_id,
// ordering_key). Chaves diferentes devem poder correr em paralelo.
type KeyLock interface {
	Lock(key string) (unlock func())
}

// SingleMutex é a alternativa ingênua medida no Capítulo 8: um mutex
// para a partição inteira. Correto — e um gargalo: chaves diferentes
// esperam umas pelas outras.
type SingleMutex struct{ mu sync.Mutex }

// Lock trava a partição inteira.
func (s *SingleMutex) Lock(string) func() {
	s.mu.Lock()
	return s.mu.Unlock
}

// livro:inicio striped

// Striped distribui as chaves por N mutexes. Operações sobre a mesma
// chave caem sempre na mesma faixa e se serializam; chaves diferentes
// quase sempre caem em faixas diferentes e correm em paralelo
// (ADR-006).
type Striped struct {
	seed   maphash.Seed
	faixas []faixa
}

// faixa ocupa uma linha de cache inteira: mutexes vizinhos na mesma
// linha disputariam o cache mesmo guardando chaves diferentes (falso
// compartilhamento).
type faixa struct {
	mu sync.Mutex
	_  [56]byte
}

// NewStriped cria n faixas; n é arredondado para potência de 2.
func NewStriped(n int) *Striped {
	p := 1
	for p < n {
		p <<= 1
	}
	return &Striped{seed: maphash.MakeSeed(), faixas: make([]faixa, p)}
}

// Lock trava a faixa da chave e devolve a função que a destrava.
func (s *Striped) Lock(key string) func() {
	f := &s.faixas[maphash.String(s.seed, key)&uint64(len(s.faixas)-1)]
	f.mu.Lock()
	return f.mu.Unlock
}

// livro:fim striped
