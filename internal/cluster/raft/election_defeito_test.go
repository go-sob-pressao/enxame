//go:build defeito

package raft_test

import (
	"testing"

	"github.com/go-sob-pressao/enxame/internal/cluster/raft/rafttest"
)

// livro:inicio teste-voto-dividido

// Sem sorteio, os três nós desistem de esperar no mesmo tick, votam em
// si mesmos, negam o voto aos outros e recomeçam — para sempre.
func TestSemSorteioNinguemVence(t *testing.T) {
	c := rafttest.New(3, 1, rafttest.Opcoes{})
	c.Run(1000)
	if l := c.Leader(); l != 0 {
		t.Skipf("houve líder (%d): o voto dividido não se repetiu", l)
	}
	t.Logf(
		"1000 ticks, termo %d, nenhum líder",
		c.Node(1).Status().Term,
	)
}

// livro:fim teste-voto-dividido
