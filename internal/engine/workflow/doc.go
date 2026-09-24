// Package workflow — avanço de workflow runs no lado do motor.
//
// Aplica o resultado de um passo ao run, agenda o próximo job de passo
// ou o timer de sleep, e encerra o run — sempre na transação atômica da
// partição.
//
// Camada: internal/engine
// Introduzido no livro: Cap. 14 e Cap. 17 — ver docs/mapa-capitulos.md
package workflow
