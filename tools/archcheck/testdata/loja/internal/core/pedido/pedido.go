// Package pedido reúne, de propósito, todas as violações de pureza.
package pedido

import (
	"os"
	"time"
	"uuid"

	"exemplo.com/externa"
	"exemplo.com/loja/internal/store"
)

// Prazo usa time.Duration: valor, permitido.
const Prazo = 5 * time.Minute

// Criar viola a pureza de cinco maneiras.
func Criar() (uuid.UUID, time.Time, int) {
	_ = os.Getenv("X")
	_ = store.Nome
	agora := time.Now()
	return uuid.NewV7(), agora.Add(Prazo), externa.Dobro(1)
}

// Relogio guarda time.Now como valor: também é proibido.
var Relogio = time.Now
