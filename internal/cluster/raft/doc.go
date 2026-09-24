// Package raft — implementação de Raft: eleição, replicação de log,
// commit e snapshot.
//
// Escrita do zero, a partir do paper, em quatro etapas testáveis.
//
// Camada: internal/cluster
// Introduzido no livro: Cap. 25 — ver docs/mapa-capitulos.md
package raft
