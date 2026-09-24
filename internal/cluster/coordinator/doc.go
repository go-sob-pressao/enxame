// Package coordinator — contrato de coordenação do cluster e sua suíte
// de conformidade.
//
// Quem está no cluster, quem é líder e qual nó possui cada partição.
// Duas implementações passam pela mesma suíte: pgcoord (produção) e
// raftcoord (a do livro). Ver ADR-003.
//
// Camada: internal/cluster
// Introduzido no livro: Cap. 23 a Cap. 26 — ver docs/mapa-capitulos.md
package coordinator
