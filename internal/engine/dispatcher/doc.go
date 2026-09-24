// Package dispatcher — pipeline de processamento por partição.
//
// Recebe jobs disponíveis, timers disparados e comandos da API, e os
// encaminha às goroutines de execução com lock striped por chave. Ver
// ADR-006.
//
// Camada: internal/engine
// Introduzido no livro: Caps. 3 a 9 — ver docs/mapa-capitulos.md
package dispatcher
