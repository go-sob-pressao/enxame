// Package caminho — a corrida que o -race não viu.
package caminho

import "errors"

// ErrTransitorio é uma falha que vale repetir.
var ErrTransitorio = errors.New("falha transitória")

// Envio é uma tentativa de entrega.
type Envio func(destino string) error
