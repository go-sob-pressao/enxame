//go:build !defeito_commit

package raft_test

import (
	"slices"
	"testing"

	"github.com/go-sob-pressao/enxame/internal/cluster/raft"
)

// livro:inicio figura-8-correto

// Com a regra de commit, A só fica comitada junto com a entrada de
// liderança do termo de S1; S5, com um log menos atualizado que o da
// maioria, não consegue os votos, e A continua onde estava.
func TestFigura8ComARegraDeCommit(t *testing.T) {
	c := figura8(t)
	semViolacoes(t, c)
	for _, id := range []raft.NodeID{2, 3} {
		temA := slices.ContainsFunc(
			c.Node(id).Log(),
			func(e raft.Entry) bool {
				return string(e.Data) == "A"
			},
		)
		if !temA {
			t.Fatalf("S%d perdeu A: %v", id, c.Node(id).Log())
		}
	}
}

// livro:fim figura-8-correto
