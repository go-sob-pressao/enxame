// Package delivery — entrega de webhooks: cliente HTTP, assinatura,
// breaker e rate limit por endpoint.
//
// A entrega é um job como outro qualquer (kind "webhook.deliver"): a
// plataforma usa a própria primitiva. O que é específico mora aqui —
// timeout agressivo, circuit breaker e token bucket por endpoint,
// registro da tentativa.
//
// Camada: internal/delivery
// Introduzido no livro: Cap. 20 e Cap. 21 — ver docs/mapa-capitulos.md
package delivery
