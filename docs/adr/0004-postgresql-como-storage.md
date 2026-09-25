# ADR 0004 — PostgreSQL como storage de produção

- **Status:** aceita
- **Capítulo:** 14
- **Data:** 2026-09-25

## Contexto

O Enxame precisa sobreviver ao processo: jobs, históricos e runs de
workflow não podem morar na memória de quem os executa. Os requisitos são
transações que cubram projeção e evento juntos, travas de linha para que
dois workers não peguem o mesmo job, e uma operação conhecida — backup,
réplica, monitoração — que a equipe do leitor já tenha.

## Opções consideradas

1. **Storage engine próprio** (um log em arquivo, com índice em memória).
   Controle total do formato e do desempenho.
2. **SQLite.** Embutido, transacional, sem servidor.
3. **PostgreSQL 18.** Servidor separado, transações, `FOR UPDATE SKIP
   LOCKED`, `JSONB`, `uuidv7()`.

## Decisão

Opção 3, atrás da interface `engine.Store` e aprovada pela mesma suíte de
contrato que a memória e o SQLite (Capítulo 12). A opção 1 foi rejeitada
pelo que exigiria antes de guardar o primeiro job: fsync correto,
recuperação de log truncado, compactação, backup — cada um, um capítulo de
bugs que o PostgreSQL já pagou. A opção 2 continua no repositório, para
testes e para o modo embutido, mas não serve a vários processos: um
escritor por vez, e sem `SKIP LOCKED`.

`SELECT … FOR UPDATE SKIP LOCKED` é o mecanismo de busca: em
`test/integration/pool_test.go`, dois pools disputam 200 jobs e cada um é
executado uma única vez, sem trava em memória nenhuma.

## Consequências

- **Melhor:** atomicidade de projeção, evento e (Capítulo 15) trabalho
  futuro no mesmo `COMMIT`; operação conhecida.
- **Pior:** uma dependência de infraestrutura para rodar em produção; cada
  transição custa idas e voltas ao banco — no Experimento 14.1, com oito
  workers e handlers de 10 ms, o poller único do pool da Parte I limita a
  vazão a cerca de 250 jobs por segundo.
- **Mais difícil de mudar depois:** as garantias que o SQL dá de graça —
  `SKIP LOCKED`, a trava de linha que ordena o `seq` — são assumidas pelo
  resto do código. Trocar de banco exige reencontrá-las.
