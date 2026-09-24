// Package cron — agendamentos periódicos (cron) com garantia de disparo
// único por janela.
//
// Cada disparo é um job com chave única derivada de schedule_id +
// instante previsto, o que torna idempotente o disparo repetido após
// falha do líder.
//
// Camada: internal/engine
// Introduzido no livro: Cap. 16 — ver docs/mapa-capitulos.md
package cron
