//go:build !defeito_eleicao

package raft_test

import (
	"testing"

	"github.com/go-sob-pressao/enxame/internal/cluster/raft"
	"github.com/go-sob-pressao/enxame/internal/cluster/raft/rafttest"
)

// livro:inicio teste-eleicao

// Em mil seeds, cinco nós elegem um líder, e só um, em até 100 ticks.
func TestEleicaoEmMilSeeds(t *testing.T) {
	for seed := range uint64(1000) {
		c := rafttest.New(5, seed, rafttest.Opcoes{})
		c.Run(100)
		semViolacoes(t, c)
		lider := c.Leader()
		if lider == 0 {
			t.Fatalf("seed %d: nenhum líder em 100 ticks", seed)
		}
		for _, id := range c.IDs() {
			if s := c.Node(id).Status(); s.Leader != lider {
				t.Fatalf(
					"seed %d: nó %d reconhece %d, líder é %d",
					seed,
					id,
					s.Leader,
					lider,
				)
			}
		}
	}
}

// livro:fim teste-eleicao

func TestReeleicaoQuandoLiderCai(t *testing.T) {
	c := rafttest.New(5, 7, rafttest.Opcoes{})
	c.Run(100)
	antigo := c.Leader()
	termoAntigo := c.Node(antigo).Status().Term
	c.Crash(antigo)
	c.Run(100)
	novo := c.Leader()
	if novo == 0 || novo == antigo ||
		c.Node(novo).Status().Term <= termoAntigo {
		t.Fatalf(
			"antigo %d (termo %d), novo %d",
			antigo,
			termoAntigo,
			novo,
		)
	}
	semViolacoes(t, c)
}

// A minoria não elege ninguém; a maioria elege um líder de termo maior.
// O antigo líder, isolado, continua se achando líder — num termo que já
// não vale. Quando a partição acaba, ele vê o termo maior e desce.
func TestParticaoMinoritaria(t *testing.T) {
	c := rafttest.New(5, 11, rafttest.Opcoes{})
	c.Run(100)
	antigo := c.Leader()
	var minoria, maioria []raft.NodeID
	minoria = append(minoria, antigo)
	for _, id := range c.IDs() {
		if id == antigo {
			continue
		}
		if len(minoria) < 2 {
			minoria = append(minoria, id)
		} else {
			maioria = append(maioria, id)
		}
	}
	c.Partition(minoria, maioria)
	c.Run(200)
	novo := c.Leader()
	if novo == antigo || !contem(maioria, novo) {
		t.Fatalf(
			"líder %d depois da partição; maioria %v",
			novo,
			maioria,
		)
	}
	c.Heal()
	c.Run(50)
	if s := c.Node(antigo).Status(); s.State == raft.Leader {
		t.Fatalf("o antigo líder %d não desceu depois da cura", antigo)
	}
	semViolacoes(t, c)
}
