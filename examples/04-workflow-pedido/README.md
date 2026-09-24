# 04 — Workflow de pedido (saga)

Reservar estoque → cobrar → emitir nota → notificar. Se a cobrança falhar,
compensa a reserva. Cada passo é memoizado: se o processo morrer entre a
cobrança e a nota, o replay não cobra de novo (ADR-002).

```go
func ProcessarPedido(ctx workflow.Context, p Pedido) error {
    reserva, err := workflow.Step(ctx, "reservar", func(ctx context.Context) (Reserva, error) {
        return estoque.Reservar(ctx, p.Itens)
    })
    if err != nil {
        return err
    }
    if _, err := workflow.Step(ctx, "cobrar", cobrar(p)); err != nil {
        workflow.Step(ctx, "liberar-reserva", liberar(reserva))
        return err
    }
    pago, _ := workflow.WaitSignal[Confirmacao](ctx, "pagamento-confirmado", 48*time.Hour)
    ...
}
```

**Capítulos 14 e 17.**
