//go:build simulation && !defeito_mapa

package simulation

import (
	"maps"
	"slices"

	"github.com/go-sob-pressao/enxame/internal/cluster/raft"
)

// livro:inicio ordem-dos-nos

// ordemDosNos devolve os nós em ordem crescente de ID. A ordem importa:
// cada nó sorteia o atraso do primeiro tick, e sorteios feitos em outra
// ordem dão outros atrasos a outros nós — outra execução.
func ordemDosNos(nos map[raft.NodeID]*raft.Node) []raft.NodeID {
	return slices.Sorted(maps.Keys(nos))
}

// livro:fim ordem-dos-nos
