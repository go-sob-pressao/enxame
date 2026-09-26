//go:build defeito_commit

package raft

// livro:inicio pode-comitar-defeito

// podeComitar, na versão com defeito: qualquer entrada que a maioria
// tenha está comitada, seja de que termo for. E o líder não grava a
// entrada de liderança.
func (n *Node) podeComitar(Index) bool { return true }

func (n *Node) entradaDeLideranca() bool { return false }

// livro:fim pode-comitar-defeito
