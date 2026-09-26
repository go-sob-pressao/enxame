# Capítulo 24 — Quem manda?

| Diretório | O que mostra |
|---|---|
| `zumbi/` | dois processos disputam a partição 0; o dono pausado com `SIGSTOP` volta e escreve — sem cerca, por cima do novo dono; com `-cerca`, recusado |

```bash
make up
export ENXAME_DB_DSN=postgres://postgres:enxame@localhost:5432/enxame?sslmode=disable
go run ./examples/cap24/zumbi preparar
go run ./examples/cap24/zumbi -no a &            # e -cerca, no Experimento 24.2
kill -STOP %1
go run ./examples/cap24/zumbi -no b &
kill -CONT %1
go run ./examples/cap24/zumbi conferir
go test -tags integration -run TestZumbi -v ./test/integration/
go test -v -run Fencing ./internal/store/postgres/     # o enigma #23
```

A posse de partição está em `internal/engine/partition`; a cerca, em
`internal/store/postgres/fencing.go`; o `pgcoord`, em
`internal/cluster/pgcoord`.
