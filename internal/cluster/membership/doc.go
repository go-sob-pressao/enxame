// Package membership — quem está vivo no cluster: batimento por nó no
// banco, com o relógio do banco, e um detector phi accrual que passa
// por um estado de suspeita antes de declarar a morte.
//
// O SWIM, que dispensa o banco, é estudado por simulação no Cap. 23
// (examples/cap23/gossip).
//
// Camada: internal/cluster
// Introduzido no livro: Cap. 23 — ver docs/mapa-capitulos.md
package membership
