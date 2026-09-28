# SLOs do Enxame

Três promessas, medidas no servidor, numa janela móvel de 30 dias
(Capítulo 32). As regras e os alarmes estão em
`deploy/docker/alertas.yml`, grupo `enxame-slo`.

| SLO | O que conta como ruim | Objetivo | Orçamento em 30 dias |
|---|---|---|---|
| `enfileirar` | `POST /v1/jobs` ou `/v1/workflows` com 5xx | 99,9% | 0,1%: 43 min com tudo falhando |
| `latencia` | `POST /v1/jobs` acima de 100 ms | 99% | 1% das requisições |
| `pontualidade` | job que começa mais de 5 s depois da hora | 99% | 1% dos jobs |

O que ficou de fora, de propósito: a duração dos jobs (é do código da
aplicação, não do Enxame), os `4xx` (erro de quem chama) e o `429` do
limite por namespace (a promessa é recusar o excesso).

## Alarmes

Um alarme por ritmo de queima, não por limiar de erro:

- **Página** — 2% do orçamento do mês em 1 hora (14,4 vezes o ritmo
  que o gasta em 30 dias), confirmado na janela de 5 minutos.
- **Ticket** — 5% em 6 horas (6 vezes), confirmado em 30 minutos.

Os testes (`make alertas`) conferem: 10% de erro vira página; 1%,
ticket; 0,1%, que gasta o orçamento no ritmo exato do mês, nada.

## Política do orçamento

- **Orçamento sobrando:** as versões saem no ritmo normal. Um
  experimento de caos em produção (Cap. 28) só com orçamento sobrando.
- **Orçamento abaixo de 25% do mês:** as versões novas passam a sair
  uma por dia, com a lista de antes de cada versão completa.
- **Orçamento esgotado:** as versões param, exceto as que consertam a
  confiabilidade ou uma vulnerabilidade, até a janela de 30 dias voltar
  a ter orçamento. A decisão é do dono do sistema (`docs/dono.md`), e
  fica registrada no canal do time.

## Antes de cada versão

O que não cabe na CI de cada commit (Cap. 31) e roda antes de cada
versão, pelo dono da versão:

1. `make chaos` — os experimentos do Capítulo 28, contra o cluster.
2. Os benchmarks do caminho quente, comparados com a versão anterior
   por `benchstat` (Cap. 29), numa máquina quieta.
3. A atualização gradual com carga — o Teste de Realidade #4 —, no
   kind, da versão em produção para a nova.
