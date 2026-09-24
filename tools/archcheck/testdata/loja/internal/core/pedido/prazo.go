package pedido

import (
	"context"
	"time"
)

// DentroDoPrazo parece puro: não chama time.Now. Quem chama é o
// context.WithTimeout, por dentro.
func DentroDoPrazo(ctx context.Context, vence time.Time) bool {
	ctx, cancel := context.WithTimeout(ctx, Prazo)
	defer cancel()
	limite, _ := ctx.Deadline()
	return !limite.After(vence)
}
