//go:build !defeito_eleicao

package raft_test

import (
	"testing"

	"github.com/go-sob-pressao/enxame/internal/cluster/raft/rafttest"
)

// livro:inicio teste-snapshot

// Um seguidor fica fora enquanto o grupo comita 100 comandos, e o líder
// compacta o log. Quando o seguidor volta, as entradas de que ele
// precisa não existem mais: só o snapshot resolve.
func TestSeguidorAtrasadoRecebeSnapshot(t *testing.T) {
	c := rafttest.New(5, 21, rafttest.Opcoes{})
	c.Run(100)
	lider := c.Leader()
	atrasado := c.IDs()[0]
	if atrasado == lider {
		atrasado = c.IDs()[1]
	}
	c.Crash(atrasado)
	propor(t, c, 100, "cmd")
	c.Run(10)
	if err := c.Compactar(lider); err != nil {
		t.Fatal(err)
	}
	if n := len(c.Node(lider).Log()); n != 0 {
		t.Fatalf("o log do líder ainda tem %d entradas", n)
	}
	c.Recover(atrasado)
	c.Run(20)
	semViolacoes(t, c)
	if c.Snapshots == 0 {
		t.Fatal(
			"o seguidor alcançou sem snapshot: o teste não exercitou nada",
		)
	}
	if n := len(c.Aplicados(atrasado)); n != 100 {
		t.Fatalf("o seguidor tem %d de 100 comandos", n)
	}
}

// livro:fim teste-snapshot

// Depois de compactar, o grupo continua aceitando e replicando
// comandos.
func TestReplicacaoDepoisDaCompactacao(t *testing.T) {
	c := rafttest.New(3, 22, rafttest.Opcoes{})
	c.Run(100)
	propor(t, c, 30, "a")
	c.Run(10)
	for _, id := range c.IDs() {
		if err := c.Compactar(id); err != nil {
			t.Fatal(err)
		}
	}
	propor(t, c, 30, "b")
	c.Run(10)
	semViolacoes(t, c)
	for _, id := range c.IDs() {
		if n := len(c.Aplicados(id)); n != 60 {
			t.Fatalf("nó %d aplicou %d de 60", id, n)
		}
	}
}
