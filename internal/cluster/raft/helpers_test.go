package raft_test

import (
	"fmt"
	"slices"
	"testing"

	"github.com/go-sob-pressao/enxame/internal/cluster/raft"
	"github.com/go-sob-pressao/enxame/internal/cluster/raft/rafttest"
)

func semViolacoes(t *testing.T, c *rafttest.Cluster) {
	t.Helper()
	if v := c.Violations(); len(v) > 0 {
		t.Fatalf("violações de segurança: %v", v)
	}
}

func contem(ids []raft.NodeID, id raft.NodeID) bool {
	return slices.Contains(ids, id)
}

func propor(t *testing.T, c *rafttest.Cluster, n int, prefixo string) {
	t.Helper()
	for i := range n {
		if err := c.Propose(fmt.Appendf(nil, "%s-%d", prefixo, i)); err != nil {
			t.Fatal(err)
		}
		c.Tick()
	}
}
