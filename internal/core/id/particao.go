package id

import "hash/fnv"

// NumParticoes é o número de partições do cluster: fixo na criação, e
// nunca muda (ADR-010).
const NumParticoes = 512

// livro:inicio particao

// Particao devolve a partição de uma chave: o FNV-1a de 32 bits dela,
// módulo NumParticoes. A mesma chave cai sempre na mesma partição, em
// qualquer nó, em qualquer versão do Enxame — por isso a função e o
// número de partições não podem mudar depois de o cluster existir.
func Particao(chave string) int {
	h := fnv.New32a()
	_, _ = h.Write([]byte(chave))
	return int(h.Sum32() % NumParticoes)
}

// livro:fim particao
