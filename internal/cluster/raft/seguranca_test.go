package raft_test

import (
	"testing"

	"github.com/go-sob-pressao/enxame/internal/cluster/raft"
	"github.com/go-sob-pressao/enxame/internal/cluster/raft/rafttest"
)

// livro:inicio figura-8

// figura8 conduz o cenário da Figura 8 do artigo do Raft com cinco nós,
// sem depender de sorteio: quem se candidata e quem ouve quem é
// roteiro. Devolve o grupo ao fim, para os testes conferirem as
// violações.
func figura8(t *testing.T) *rafttest.Cluster {
	t.Helper()
	// Timeout de eleição enorme: só há eleição quando o roteiro manda.
	c := rafttest.New(5, 1, rafttest.Opcoes{ElectionTicks: 1_000_000})
	um, dois, tres, quatro, cinco := raft.NodeID(1), raft.NodeID(2),
		raft.NodeID(3), raft.NodeID(4), raft.NodeID(5)

	// (a) S1 lidera e grava A, que chega só a S2.
	c.Campaign(um)
	c.Partition(
		[]raft.NodeID{um, dois},
		[]raft.NodeID{tres, quatro, cinco},
	)
	c.ProposeTo(um, []byte("A"))

	// (b) S1 cai. S5 é eleito por S3 e S4, grava B e cai antes de
	// replicar qualquer coisa: nenhum AppendEntries dele chega.
	c.Crash(um)
	c.Bloquear(cinco, raft.MsgApp)
	c.Campaign(cinco)
	c.ProposeTo(cinco, []byte("B"))
	c.Crash(cinco)

	// (c) S1 volta, é eleito por S2 e S3 e replica A para S3: A está na
	// maioria (S1, S2, S3).
	c.Recover(um)
	c.Partition([]raft.NodeID{um, dois, tres}, []raft.NodeID{quatro})
	c.Campaign(um)
	c.Campaign(um) // S3 votou em S5 no termo 2: S1 precisa do termo 3
	c.Run(3)

	// (d) S1 cai. S5 volta e se candidata com o log dele (B, de um
	// termo mais recente que A).
	c.Crash(um)
	c.Recover(cinco)
	c.Heal()
	c.Campaign(cinco)
	c.Campaign(
		cinco,
	) // a primeira tentativa só descobre o termo corrente
	c.Run(10)
	return c
}

// livro:fim figura-8
