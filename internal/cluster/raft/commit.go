package raft

// livro:inicio commit

// maybeCommit avança o commit até o maior índice que a maioria já tem e
// que o líder pode comitar por contagem (ver podeComitar).
func (n *Node) maybeCommit() {
	for i := n.lastIndex(); i > n.commit; i-- {
		if n.podeComitar(i) && n.replicadas(i) >= n.quorum() {
			n.commit = i
			return
		}
	}
}

// livro:fim commit

// replicadas conta quantos membros (inclusive o líder) já têm a entrada
// i.
func (n *Node) replicadas(i Index) int {
	total := 0
	for _, p := range n.cfg.Peers {
		if n.match[p] >= i {
			total++
		}
	}
	return total
}
