// Package timer — timer pump durável: jobs agendados, retries e sleeps
// de workflow.
//
// Min-heap em memória com horizonte de lookahead, alimentado pelo
// banco. O relógio é injetado — requisito da simulação determinística.
//
// Camada: internal/engine
// Introduzido no livro: Cap. 11 e Cap. 27 — ver docs/mapa-capitulos.md
package timer
