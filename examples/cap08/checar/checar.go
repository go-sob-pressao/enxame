// Package checar — o cache de conexões que abria duas conexões por
// endpoint.
package checar

import "sync/atomic"

// Conexao simula uma conexão cara de abrir.
type Conexao struct{ Endpoint string }

// abertas conta quantas conexões foram abertas de verdade.
var abertas atomic.Int64

func abrir(endpoint string) *Conexao {
	abertas.Add(1)
	return &Conexao{Endpoint: endpoint}
}

// Abertas devolve e zera o contador.
func Abertas() int64 { return abertas.Swap(0) }
