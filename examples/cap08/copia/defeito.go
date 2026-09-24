//go:build defeito

package copia

import "sync"

// livro:inicio copia-defeito

// Estoque parece protegido. Mas Reservar tem receptor de VALOR: cada
// chamada trava uma cópia do Mutex. O vet acusa; o compilador, não.
type Estoque struct {
	mu        sync.Mutex
	reservado *int
}

// Reservar reserva q unidades.
func (e Estoque) Reservar(q int) {
	e.mu.Lock()
	defer e.mu.Unlock()
	*e.reservado += q
}

// livro:fim copia-defeito
