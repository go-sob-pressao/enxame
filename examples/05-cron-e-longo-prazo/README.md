# 05 — Cron

Um agendamento a cada minuto, no fuso de São Paulo, disparado pelo worker
embutido. Cada disparo é um job com chave única por janela: o disparo
repetido — um reinício, uma regravação do agendamento, dois agendadores — é
absorvido pelo banco.

```go
client.Schedule(ctx, enxame.Schedule{
    ID: "fechamento", Cron: "* * * * *", Timezone: "America/Sao_Paulo",
    Args: FecharCaixa{Loja: "centro"},
})
```

```bash
go run ./examples/05-cron-e-longo-prazo   # espera o próximo minuto cheio
```

**Capítulos 16 e 17.** Para processos de longo prazo, um workflow com
`workflow.Sleep` de dias sobrevive a deploys e reinícios (exemplo 04).
