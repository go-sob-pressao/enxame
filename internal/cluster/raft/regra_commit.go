//go:build !defeito_commit

package raft

// livro:inicio pode-comitar

// podeComitar: o líder só comita POR CONTAGEM entradas do próprio
// termo. As de termos anteriores ficam comitadas indiretamente, quando
// uma entrada do termo corrente depois delas chega à maioria. Contar
// réplicas de uma entrada antiga não basta: um líder futuro, eleito com
// votos de quem não a tem, pode sobrescrevê-la (Figura 8 do artigo do
// Raft).
func (n *Node) podeComitar(i Index) bool {
	t, _ := n.termAt(i)
	return t == n.term
}

// entradaDeLideranca: ao assumir, o líder grava uma entrada vazia do
// próprio termo. Quando ela chega à maioria, tudo o que veio antes fica
// comitado junto — sem esperar a aplicação propor algo.
func (n *Node) entradaDeLideranca() bool { return true }

// livro:fim pode-comitar
