//go:build defeito

package raft

// livro:inicio timeout-defeito

// sortearTimeout, na versão com defeito, não sorteia nada: todos os nós
// esperam exatamente ElectionTicks.
func (n *Node) sortearTimeout() int {
	return n.cfg.ElectionTicks
}

// livro:fim timeout-defeito
