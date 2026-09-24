# 02 — Enfileiramento transacional

O pedido e o job de cobrança entram no **mesmo COMMIT**. Ou os dois existem,
ou nenhum — não há janela de dual-write.

```go
tx, _ := db.Begin(ctx)
defer tx.Rollback(ctx)

pedidoID, _ := pedidos.Criar(ctx, tx, p)
client.InsertTx(ctx, tx, CobrarPedido{PedidoID: pedidoID},
    enxame.UniqueKey("cobranca:"+pedidoID))

tx.Commit(ctx)
```

É o padrão outbox transformado em recurso de produto. **Capítulo 15.**
O exemplo abre com a versão errada (grava no banco e publica numa fila) e a
falha injetada que perde a cobrança.
