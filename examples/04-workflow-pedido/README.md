# 04 — Workflow de pedido (saga)

Reservar estoque → cobrar → esperar o prazo de arrependimento → emitir a
nota. Se a cobrança for recusada, compensa a reserva. Cada passo é memoizado:
se o processo morrer entre a cobrança e a nota, o replay não cobra de novo
(ADR-002), e a cobrança recebe a mesma chave de idempotência em cada
tentativa (`job.IdempotencyKey`).

```go
w := client.NewWorker(enxame.WorkerConfig{})
w.Workflow("pedido", ProcessarPedido)
client.StartWorkflow(ctx, "pedido", pedido.ID, pedido)
```

```bash
go run ./examples/04-workflow-pedido
```

**Capítulos 14, 15 e 17.** Sinais (`WaitSignal`) ainda não existem no runtime.
