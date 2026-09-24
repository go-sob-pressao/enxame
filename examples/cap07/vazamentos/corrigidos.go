package vazamentos

import (
	"context"
	"time"
)

// livro:inicio corrige-envio

// PrimeiroResultadoCorrigido usa buffer do tamanho das réplicas: todo
// envio completa, mesmo que só o primeiro seja lido.
func PrimeiroResultadoCorrigido(replicas []func() int) int {
	c := make(chan int, len(replicas))
	for _, r := range replicas {
		go func() { c <- r() }()
	}
	return <-c
}

// livro:fim corrige-envio

// livro:inicio corrige-contexto

// VigiarCorrigido termina quando o contexto termina.
func VigiarCorrigido(ctx context.Context, verificar func()) {
	go func() {
		t := time.NewTicker(10 * time.Millisecond)
		defer t.Stop()
		for {
			verificar()
			select {
			case <-t.C:
			case <-ctx.Done():
				return
			}
		}
	}()
}

// livro:fim corrige-contexto
