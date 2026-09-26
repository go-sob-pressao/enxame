// Package cron — o laço que dispara os agendamentos periódicos no modo
// servidor. A expressão, a janela e a decisão de cada disparo são do
// domínio, em internal/core/schedule.
//
// Cada disparo é um job com chave única derivada de schedule_id +
// instante previsto, o que torna idempotente o disparo repetido após
// falha do líder.
//
// Camada: internal/engine
// Introduzido no livro: Cap. 16 — ver docs/mapa-capitulos.md
package cron
