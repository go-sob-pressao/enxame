// Package tracing — tracing distribuído com OpenTelemetry.
//
// O contexto de trace viaja gravado no job, na coluna trace_parent, e
// atravessa o banco, reinícios e dias de espera entre os passos de um
// workflow.
//
// Camada: internal/observ
// Introduzido no livro: Cap. 30 — ver docs/mapa-capitulos.md
package tracing
