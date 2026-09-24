//go:build !defeito

package ctxsombreado

import (
	"context"

	"golang.org/x/sync/errgroup"
)

// livro:inicio ctx-sombreado-correto

// BuscarTodos reatribui ctx ao contexto do grupo: não sobra um ctx "de
// fora" para usar por engano.
func BuscarTodos(ctx context.Context, n int) error {
	g, ctx := errgroup.WithContext(ctx)
	for i := range n {
		g.Go(func() error { return Buscar(ctx, i == 0) })
	}
	return g.Wait()
}

// livro:fim ctx-sombreado-correto
