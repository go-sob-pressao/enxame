// Package valor é domínio limpo: importar outro pacote do core é permitido.
package valor

import (
	"time"

	"exemplo.com/loja/internal/core/pedido"
)

// Vence calcula o vencimento a partir de um instante recebido.
func Vence(t time.Time) time.Time { return t.Add(pedido.Prazo) }
