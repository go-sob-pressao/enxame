# Capítulo 21 — Backpressure e rate limiting

| Diretório | O que mostra |
|---|---|
| `retentar/` | o enigma: um canal limitado a 10 mil ao lado de uma lista de retentativas sem limite — e a correção, com o limite no sistema inteiro |

O gerador de carga em malha aberta está em `test/load`:

```bash
make up
go run ./cmd/enxamed -max-em-curso 0 &          # sem descarte
go run ./test/load -token t1 -fases 5000:10s    # Custo Real #7
go run ./test/load -token t1 \
    -fases 500:10s,10000:20s,500:30s -csv tempo.csv
go test -tags integration -run Realidade2 ./test/integration/
go test -v ./examples/cap21/retentar/
```

O limite por namespace e o descarte estão em
`internal/transport/http/limite.go`; o balde de fichas, em
`internal/core/policy/tokenbucket.go`; o limite por endpoint e o
`Retry-After` da entrega, em `internal/delivery/entrega.go`.
