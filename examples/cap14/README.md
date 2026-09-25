# Capítulo 14 — Estado que sobrevive

| Diretório | O que mostra |
|---|---|
| `auditoria/` | a alternativa rejeitada pela ADR-001: estado mutável com auditoria ao lado, e o resgate que muda o estado sem registrar (`-tags defeito`) |
| `lacuna/` | o enigma: o leitor que anda por um id de sequence e pula o evento cuja transação fez COMMIT por último (`-tags defeito`); a mesma intercalação com `next_seq` travado, que não pula nada. Precisa de `ENXAME_DB_DSN` |
| `sobrevive/` | o Experimento 14.1: `kill -9` no meio do trabalho, conferência do histórico e resgate |

```bash
make up
export ENXAME_DB_DSN='postgres://postgres:enxame@localhost:5432/enxame?sslmode=disable'
go test -tags defeito ./examples/cap14/...
go run ./examples/cap14/sobrevive enfileirar 5000
```

A fila no Postgres, o runtime de workflow e o benchmark de replay são código
do Enxame: `internal/store/postgres/`, `pkg/workflow/`,
`internal/worker/workflow/` e `test/integration/replay_bench_test.go`.
