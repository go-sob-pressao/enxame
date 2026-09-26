# 03 — Webhooks

A aplicação publica um evento na própria transação (`PublishTx`); o Enxame o
entrega a cada endpoint inscrito, assinado no formato Standard Webhooks, com
prazo, retry com jitter, circuit breaker por endpoint e registro de cada
tentativa.

```go
client.CreateEndpoint(ctx, enxame.Endpoint{
    URL: "https://cliente.exemplo/hooks", EventTypes: []string{"pedido.pago"},
    SecretRef: "env:SEGREDO_CLIENTE"})
client.PublishTx(ctx, tx, webhook.Message{EventType: "pedido.pago", Payload: p})
```

Do lado de quem recebe:

```go
id, err := webhook.Verificar(segredo, r.Header, corpo, time.Now())
```

```bash
go run ./examples/03-webhooks   # o receptor fica fora do ar por 6 s
```

**Capítulos 15, 16 e 20.**
