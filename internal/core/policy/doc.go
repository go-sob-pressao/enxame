// Package policy — políticas de retry, backoff com jitter, timeout,
// rate limit e retenção.
//
// O cálculo do próximo retry recebe a fonte de aleatoriedade como
// parâmetro: a política é pura e o jitter continua reproduzível na
// simulação.
//
// Camada: internal/core Introduzido no livro: Cap. 16, Cap. 20 e Cap.
// 21 — ver docs/mapa-capitulos.md
package policy
