// Package job — máquina de estados do job, transições e invariantes de
// domínio.
//
// available → running → completed | retryable | discarded | cancelled.
// Toda transição é uma função pura: recebe o estado e o evento, devolve
// o novo estado e os eventos a persistir. Sem I/O, sem relógio.
//
// Camada: internal/core Introduzido no livro: Cap. 2 (M0) e Cap. 10
// (TDD) — ver docs/mapa-capitulos.md
package job
