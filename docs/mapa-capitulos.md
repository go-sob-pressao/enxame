# Mapa capítulo → diretório

| Parte | Cap. | O que é construído | Diretórios |
|---|---|---|---|
| 0 | 0–2 | domínio puro, esqueleto, archcheck | `internal/core/`, `cmd/`, `tools/archcheck/` |
| I | 3–9 | pool de workers, filas, long-poll, dispatcher | `internal/worker/`, `internal/queue/`, `internal/engine/dispatcher/` |
| II | 10–13 | TDD do domínio, synctest, clock injetável, store em memória, suíte de contrato, fuzzing | `internal/core/`, `internal/store/memory/`, `internal/simulation/clock/`, `test/testutil/` |
| III | 14–17 | Postgres, histórico, enfileiramento transacional, idempotência, cron, workflows, migrações | `internal/store/`, `internal/engine/`, `pkg/enxame/`, `pkg/workflow/` |
| IV | 18–21 | gRPC e workers remotos, API HTTP, webhooks, resiliência, rate limit | `internal/transport/`, `internal/delivery/`, `pkg/webhook/` |
| V | 22–26 | membership, pgcoord, Raft e raftcoord, particionamento e ordem por chave | `internal/cluster/` |
| VI | 27–29 | simulação, caos, desempenho | `internal/simulation/`, `test/simulation/`, `test/chaos/` |
| VII | 30–32 | observabilidade, entrega, operação | `internal/observ/`, `deploy/` |
