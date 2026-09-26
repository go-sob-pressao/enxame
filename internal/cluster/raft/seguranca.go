package raft

// livro:inicio log-atualizado

// logAtualizado é a restrição de eleição: o candidato só recebe o voto
// se o log dele for pelo menos tão atualizado quanto o de quem vota —
// último termo maior, ou mesmo último termo e log pelo menos tão longo.
// Como toda entrada comitada está numa maioria, e o candidato precisa
// de uma maioria de votos, alguém dessa maioria tem a entrada e nega o
// voto a quem não tem: o líder eleito sempre tem todas as comitadas.
func (n *Node) logAtualizado(m Message) bool {
	if m.LastTerm != n.lastTerm() {
		return m.LastTerm > n.lastTerm()
	}
	return m.LastIndex >= n.lastIndex()
}

// livro:fim log-atualizado
