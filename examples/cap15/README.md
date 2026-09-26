# Capítulo 15 — Transações e o padrão outbox

| Diretório | O que mostra |
|---|---|
| `dualwrite/` | gravar e publicar em dois sistemas: o COMMIT seguido de publicação que perde a cobrança, e o enigma — gravar, publicar e comitar, nessa ordem (`-tags defeito`); a versão com `InsertTx`, tudo ou nada |
| `perdida/` | a atualização perdida em READ COMMITTED, e as três saídas: somar no banco, REPEATABLE READ e lock otimista com `version` |
| `custo/` | o Custo Real #4: uma transação, duas transações e dual-write, com uma e oito goroutines |

Todos precisam de `ENXAME_DB_DSN` (o de `make up`); sem ele, os testes são
pulados.

```bash
go test -tags defeito ./examples/cap15/dualwrite/
go test ./examples/cap15/...
go test -run '^$' -bench . -cpu 1,8 ./examples/cap15/custo/
go run ./examples/02-enfileiramento-transacional
```

A transação atômica do workflow está em
`internal/store/postgres/transicao.go`; o cliente público, em `pkg/enxame`.
