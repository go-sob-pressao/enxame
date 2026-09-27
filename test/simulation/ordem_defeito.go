//go:build simulation && defeito_mapa

package simulation

import "github.com/go-sob-pressao/enxame/internal/cluster/raft"

// livro:inicio ordem-dos-nos-defeito

// ordemDosNos, na versão com defeito, percorre o map. A ordem de
// iteração de um map em Go é deliberadamente variável entre execuções.
func ordemDosNos(nos map[raft.NodeID]*raft.Node) []raft.NodeID {
	var ids []raft.NodeID
	for id := range nos {
		ids = append(ids, id)
	}
	return ids
}

// livro:fim ordem-dos-nos-defeito
