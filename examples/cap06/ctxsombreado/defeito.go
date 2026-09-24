//go:build defeito

package ctxsombreado

import (
	"context"

	"golang.org/x/sync/errgroup"
)

// livro:inicio ctx-sombreado-defeito

// BuscarTodos deveria cancelar as buscas restantes quando uma falha. O
// errgroup cancela o ctx DERIVADO — mas as goroutines usam o de fora.
func BuscarTodos(ctx context.Context, n int) error {
	g, gctx := errgroup.WithContext(ctx)
	_ = gctx
	for i := range n {
		g.Go(func() error { return Buscar(ctx, i == 0) })
	}
	return g.Wait()
}

// livro:fim ctx-sombreado-defeito
