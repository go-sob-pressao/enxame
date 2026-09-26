package raft_test

import (
	"fmt"
	"testing"

	"github.com/go-sob-pressao/enxame/internal/cluster/raft"
	"github.com/go-sob-pressao/enxame/internal/cluster/raft/rafttest"
)

func propor(t *testing.T, c *rafttest.Cluster, n int, prefixo string) {
	t.Helper()
	for i := range n {
		if err := c.Propose(fmt.Appendf(nil, "%s-%d", prefixo, i)); err != nil {
			t.Fatal(err)
		}
		c.Tick()
	}
}

// livro:inicio teste-replicacao

// Cem comandos propostos ao líder chegam, na mesma ordem, a todos.
func TestReplicacaoEmOrdem(t *testing.T) {
	c := rafttest.New(5, 3, rafttest.Opcoes{})
	c.Run(100)
	propor(t, c, 100, "cmd")
	c.Run(20)
	semViolacoes(t, c)
	for _, id := range c.IDs() {
		if n := len(c.Aplicados(id)); n != 100 {
			t.Fatalf("nó %d aplicou %d comandos, esperado 100", id, n)
		}
	}
}

// livro:fim teste-replicacao

// Um seguidor fora do ar durante 50 comandos alcança os outros ao
// voltar: o líder recua next até o ponto em que os logs coincidem.
func TestSeguidorAtrasadoAlcanca(t *testing.T) {
	c := rafttest.New(5, 5, rafttest.Opcoes{})
	c.Run(100)
	atrasado := c.IDs()[0]
	if atrasado == c.Leader() {
		atrasado = c.IDs()[1]
	}
	c.Crash(atrasado)
	propor(t, c, 50, "a")
	c.Recover(atrasado)
	c.Run(50)
	semViolacoes(t, c)
	if n := len(c.Aplicados(atrasado)); n != 50 {
		t.Fatalf("o nó que voltou aplicou %d de 50", n)
	}
}

// Um líder isolado aceita um comando que nunca comita. A maioria elege
// outro líder e segue. Quando a partição acaba, o log do antigo líder é
// corrigido: a entrada não comitada dele é descartada.
func TestEntradaNaoComitadaDoLiderIsolado(t *testing.T) {
	c := rafttest.New(5, 9, rafttest.Opcoes{})
	c.Run(100)
	antigo := c.Leader()
	var outros []raft.NodeID
	for _, id := range c.IDs() {
		if id != antigo {
			outros = append(outros, id)
		}
	}
	c.Partition([]raft.NodeID{antigo}, outros)
	if _, err := c.Node(antigo).Propose([]byte("perdida")); err != nil {
		t.Fatal(err)
	}
	c.Run(200)
	propor(t, c, 10, "maioria")
	c.Heal()
	c.Run(100)
	semViolacoes(t, c)
	for _, e := range c.Node(antigo).Log() {
		if string(e.Data) == "perdida" {
			t.Fatal(
				"a entrada não comitada do líder isolado sobreviveu",
			)
		}
	}
	if n := len(c.Aplicados(antigo)); n != 10 {
		t.Fatalf("o antigo líder aplicou %d de 10", n)
	}
}
