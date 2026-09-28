# Missão #7 — Três da manhã

**Chamado #9204, prioridade crítica.** Às 3h12, o alarme
`FilaEnvelhecendo` tocou para a fila `relatorios`: o fechamento do mês,
que sai todo dia 1º às 3h, não saiu. A CPU dos workers está normal, a
API não tem erro, o banco está folgado. Houve um deploy do worker de
relatórios às 2h50, que "só mexeu numa configuração". O plantão tem a
telemetria da noite.

```bash
make up
git checkout missao-07-tres-da-manha
go test -count=1 -v -run TestMissao ./missoes/07-tres-da-manha/
```

O teste reproduz a noite em 25 segundos: 20 relatórios enfileirados, o
worker do deploy rodando. Na tag `missao-07`, nenhum fica pronto. Seja
qual for o resultado, a telemetria da noite fica em
`missoes/07-tres-da-manha/telemetria/`: `metricas.prom`, `spans.jsonl`
e `logs.jsonl`.

Investigue **pela telemetria** — alarme, métrica, trace, log —, antes
de ler o código, e anote o percurso. Depois conserte mudando **só**
`worker.go`.

**Critério:** `TestMissao` verde, com os 20 relatórios prontos. Dicas
em três níveis e o gabarito comentado estão no fechamento da Parte VII.
