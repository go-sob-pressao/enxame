# Enxame

Jobs, agendamentos, webhooks e workflows duráveis sobre PostgreSQL, escritos em Go.

Este é o projeto construído ao longo do livro **Go Sob Pressão — Concorrência,
Sistemas Distribuídos e Engenharia de Produção**.

> Grave o pedido e enfileire a cobrança no mesmo `COMMIT`. O Enxame garante que
> ela execute até o fim — com retry, sem cobrar duas vezes — mesmo que o
> processo morra, o nó caia ou a rede particione.

## O que ele faz

| Recurso | Em uma linha |
|---|---|
| **Jobs** | Handlers tipados, filas com prioridade, retry com backoff e jitter, chave de idempotência estável |
| **Enfileiramento transacional** | `InsertTx` grava o job na mesma transação dos seus dados — sem dual-write |
| **Agendamentos** | Cron com fuso horário e disparo único por janela, mesmo com troca de líder |
| **Webhooks** | Entrega assinada (Standard Webhooks), ordem por endpoint, circuit breaker, registro de tentativas |
| **Workflows** | Funções Go com passos memoizados: `Step`, `Sleep`, `SideEffect`, `Now`, `Version` — sobrevivem a deploys |
| **Cluster** | Partições com fencing token; coordenação por Postgres ou pelo Raft do livro |

Dois modos de uso: **biblioteca** (os workers rodam dentro da sua aplicação,
contra o seu Postgres) e **servidor** (`enxamed` em cluster, com workers remotos
via gRPC e API HTTP).

## Navegando pelo repositório

O histórico é organizado por capítulo. Cada tag corresponde ao estado do
sistema ao final daquele capítulo:

```bash
git checkout cap-07        # entra direto no capítulo 7
git diff cap-24 cap-25     # vê exatamente o que o capítulo 25 mudou
```

As missões ficam em branches próprias, com o teste já vermelho:

```bash
git checkout missao-05-lider-perdido
go test ./... -run TestMissao
```

## Começando

Requer **Go 1.27** ou superior (a versão exata está no `go.mod`) e Docker.

```bash
make up          # Postgres 18, Jaeger, Prometheus e Grafana
make build       # bin/enxamed e bin/enxamectl
make check       # lint + arquitetura + testes com -race
```

A `v1.0.0` é a versão do fim do livro; as mudanças estão em
[CHANGELOG.md](CHANGELOG.md), com o que a compatibilidade cobre.

## Em produção

- `deploy/k8s/` — o `StatefulSet` e os overlays `dev` (kind) e `prod`;
  `deploy/helm/enxame` — o mesmo, em chart
- `docs/checklist-producao.md` — os 40 itens, e o estado do Enxame em cada um
- `docs/slo.md`, `docs/runbook/`, `docs/dono.md` — o que se promete, o que se
  faz quando um alarme toca, e quem responde

## Verificações

```bash
make race        # testes com race detector (Pacto 3)
make arch        # regras de dependência entre camadas
make vuln        # govulncheck
make sim         # simulação determinística (a partir do Cap. 27)
make help        # todos os alvos
```

## Documentação

- `docs/adr/` — decisões de arquitetura registradas
- `docs/protocol/` — catálogo de eventos
- `docs/runbook/` — um runbook por alarme
- `docs/postmortem/` — o modelo e um exemplo
- `docs/mapa-capitulos.md` — o que cada capítulo constrói

## Licença

Código sob [Apache-2.0](LICENSE): você pode usar o Enxame e os trechos do livro
no seu trabalho, inclusive comercialmente. O texto do livro não faz parte deste
repositório. Correções posteriores à publicação estão em [ERRATA.md](ERRATA.md).
