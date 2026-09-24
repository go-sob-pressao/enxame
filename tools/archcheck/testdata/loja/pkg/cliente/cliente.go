// Package cliente é API pública e não pode depender do motor.
package cliente

import "exemplo.com/loja/internal/engine"

// Versao expõe, indevidamente, um detalhe do motor.
const Versao = engine.Versao
