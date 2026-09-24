# 05 — Cron e processos de longo prazo

Um agendamento diário de cobrança recorrente e um workflow de assinatura que
dorme 30 dias entre ciclos — sobrevivendo a deploys, reinícios e troca de líder.

```go
client.Schedule(ctx, "cobranca-diaria", "0 6 * * *", CobrarRecorrentes{},
    enxame.Timezone("America/Sao_Paulo"), enxame.Overlap(enxame.Skip))
```

Cada disparo é um job com chave única por janela: o disparo repetido após
falha do líder é absorvido pelo banco. **Capítulos 16 e 26.**
