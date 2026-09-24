// Package raftcoord — coordenação sobre o Raft implementado no livro.
//
// Mapa partição→nó replicado por consenso, sem depender do banco para o
// plano de controle. Comparado com pgcoord por medição no Cap. 25.
//
// Camada: internal/cluster
// Introduzido no livro: Cap. 25 — ver docs/mapa-capitulos.md
package raftcoord
