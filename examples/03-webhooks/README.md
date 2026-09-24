# 03 — Webhooks

A aplicação publica um evento; o Enxame entrega a cada endpoint inscrito,
assinado (Standard Webhooks), com retry, ordem por endpoint, circuit breaker e
registro de cada tentativa.

```go
client.PublishTx(ctx, tx, webhook.Message{
    EventType:      "pedido.pago",
    Payload:        pedido,
    IdempotencyKey: "pedido.pago:" + pedido.ID,
})
```

Do lado de quem recebe:

```go
payload, err := webhook.Verify(secret, r.Header, body)
```

**Capítulos 16, 20 e 21.** O Capítulo 20 abre com o endpoint do cliente fora do
ar e o retry sem jitter que o derruba de vez quando ele volta.
