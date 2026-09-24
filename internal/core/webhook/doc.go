// Package webhook — endpoint, mensagem, entrega e assinatura de
// webhooks.
//
// Regras puras: quais endpoints recebem qual tipo de evento, cálculo da
// assinatura HMAC-SHA256 no formato Standard Webhooks e a política de
// desativação de endpoint após falhas persistentes.
//
// Camada: internal/core
// Introduzido no livro: Cap. 16 e Cap. 20 — ver docs/mapa-capitulos.md
package webhook
