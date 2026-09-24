//go:build !defeito

package caminho

import (
	"errors"
	"sync"
)

// livro:inicio caminho-correto

// Enviador protege os dois contadores com a mesma trava.
type Enviador struct {
	mu         sync.Mutex
	entregues  int
	repeticoes int
}

// EnviarTodos envia para todos os destinos em paralelo.
func (e *Enviador) EnviarTodos(destinos []string, enviar Envio) {
	var wg sync.WaitGroup
	for _, d := range destinos {
		wg.Go(func() {
			repetiu := false
			if err := enviar(d); errors.Is(err, ErrTransitorio) {
				repetiu = true
				_ = enviar(d)
			}
			e.mu.Lock()
			defer e.mu.Unlock()
			e.entregues++
			if repetiu {
				e.repeticoes++
			}
		})
	}
	wg.Wait()
}

// livro:fim caminho-correto
