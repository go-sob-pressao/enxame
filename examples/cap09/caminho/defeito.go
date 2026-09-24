//go:build defeito

package caminho

import (
	"errors"
	"sync"
)

// livro:inicio caminho-defeito

// Enviador entrega mensagens em paralelo e conta as repetições. O
// caminho feliz não toca estado compartilhado; o de retry, sim — sem
// trava.
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
			if err := enviar(d); errors.Is(err, ErrTransitorio) {
				e.repeticoes++ // só roda quando alguém falha
				_ = enviar(d)
			}
			e.mu.Lock()
			e.entregues++
			e.mu.Unlock()
		})
	}
	wg.Wait()
}

// livro:fim caminho-defeito
