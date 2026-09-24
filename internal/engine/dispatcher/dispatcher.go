package dispatcher

import (
	"context"

	"golang.org/x/sync/errgroup"
)

// Runner é um pool de uma fila, visto pelo dispatcher.
type Runner interface {
	Run(ctx context.Context) error
}

// livro:inicio dispatcher

// Dispatcher roda um pool por fila, com concorrência própria: a fila de
// e-mails lenta não ocupa as vagas da fila de pagamentos. Se um pool
// falha, todos param — o errgroup cancela o contexto compartilhado.
type Dispatcher struct {
	Pools map[string]Runner
}

// Run executa todos os pools até o contexto terminar ou um deles
// falhar.
func (d *Dispatcher) Run(ctx context.Context) error {
	g, ctx := errgroup.WithContext(ctx)
	for _, p := range d.Pools {
		g.Go(func() error { return p.Run(ctx) })
	}
	return g.Wait()
}

// livro:fim dispatcher
