# Capítulo 17 — Migração sem downtime

| Diretório | O que mostra |
|---|---|
| `expand/` | uma coluna que muda de tipo em três deploys — expand, backfill em lotes, contract — com as três versões do código rodando ao lado da anterior, e a volta do contract. Precisa de `ENXAME_DB_DSN` |
| `versao/` | o enigma: o passo inserido no meio do workflow que quebra os runs em andamento (`-tags defeito`); a correção com `workflow.Version`; e o campo renomeado que volta vazio no replay |
| `realidade/` | o Teste de Realidade #1: 500 jobs e 50 workflows, `kill -9` no meio, nenhum perdido, nenhum efeito duplicado. Rodado por `test/integration/realidade1_test.go` |

```bash
go test ./examples/cap17/...
go test -tags defeito ./examples/cap17/versao/
go test -tags integration -run 'Realidade1|Migracoes' -v ./test/integration/
```
