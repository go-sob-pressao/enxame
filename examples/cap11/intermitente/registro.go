// Package intermitente — o teste que espera com time.Sleep.
package intermitente

import "sync"

// livro:inicio registro

// Registro guarda eventos de auditoria. Notificar grava numa goroutine
// para não atrasar quem chama — e devolve antes de gravar.
type Registro struct {
	mu      sync.Mutex
	eventos []string
}

// Notificar registra o evento em segundo plano.
func (r *Registro) Notificar(ev string) {
	go func() {
		r.mu.Lock()
		defer r.mu.Unlock()
		r.eventos = append(r.eventos, ev)
	}()
}

// Total devolve quantos eventos foram gravados até agora.
func (r *Registro) Total() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.eventos)
}

// livro:fim registro
