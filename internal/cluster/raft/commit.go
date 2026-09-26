package raft

// livro:inicio commit-ingenuo

// maybeCommit avança o commit até o maior índice que a maioria já tem.
// É a regra ingênua da etapa 2; a etapa 3 mostra por que ela perde
// entradas comitadas.
func (n *Node) maybeCommit() {
	for i := n.lastIndex(); i > n.commit; i-- {
		if n.replicadas(i) >= n.quorum() {
			n.commit = i
			return
		}
	}
}

// livro:fim commit-ingenuo

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
