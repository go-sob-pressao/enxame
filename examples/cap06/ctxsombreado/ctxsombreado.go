// Package ctxsombreado — o errgroup que não cancelava.
package ctxsombreado

import "context"

// Buscar simula uma chamada que respeita o contexto.
func Buscar(ctx context.Context, falhar bool) error {
	if falhar {
		return context.DeadlineExceeded
	}
	<-ctx.Done()
	return ctx.Err()
}
