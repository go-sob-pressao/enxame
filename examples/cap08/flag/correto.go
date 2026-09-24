//go:build !defeito

package flag

import "sync/atomic"

// livro:inicio flag-correto

// ComAtomic usa atomic.Bool: a escrita em Parar acontece-antes da
// leitura que a observa, e o compilador não pode tirar a leitura do
// laço.
type ComAtomic struct{ parar atomic.Bool }

// Parar pede que o trabalho termine.
func (c *ComAtomic) Parar() { c.parar.Store(true) }

// Trabalhar gira até parar.
func (c *ComAtomic) Trabalhar(n *int) {
	for !c.parar.Load() {
		*n++
	}
}

// livro:fim flag-correto

// Novo devolve a versão correta.
func Novo() Trabalhador { return &ComAtomic{} }
