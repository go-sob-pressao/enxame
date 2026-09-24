// Package invariant — invariantes verificadas ao final de cada cenário.
//
// Nenhum job perdido, nenhum efeito duplicado, um único dono por
// partição, ordem por chave preservada, histórico monotônico, log
// matching do Raft.
//
// Camada: internal/simulation
// Introduzido no livro: Cap. 27 — ver docs/mapa-capitulos.md
package invariant
