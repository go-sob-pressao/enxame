//go:build !defeito_eleicao

package raft

// livro:inicio timeout

// sortearTimeout escolhe o timeout de eleição em [E, 2E). O sorteio é o
// que quebra a simetria: se todos os seguidores desistissem do líder no
// mesmo tick, todos se candidatariam juntos, dividiriam os votos e
// repetiriam isso para sempre.
func (n *Node) sortearTimeout() int {
	return n.cfg.ElectionTicks + n.cfg.Rand(n.cfg.ElectionTicks)
}

// livro:fim timeout
