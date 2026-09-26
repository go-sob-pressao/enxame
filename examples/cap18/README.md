# Capítulo 18 — gRPC além do "hello"

| Diretório | O que mostra |
|---|---|
| `remoto/` | o Experimento 18.1: o motor num processo, workers remotos em outros; `kill -9` num worker e o resgate pelo motor. Precisa de `ENXAME_DB_DSN` |
| `keepalive/` | o enigma: o worker com ping a cada 10 s que o servidor padrão derruba com GOAWAY `too_many_pings` (`-tags defeito`, ~40 s); a política alinhada (~45 s) |
| `custo/` | o Custo Real #5: o mesmo batimento por gRPC, com e sem interceptors, e por HTTP com JSON |

```bash
go run ./examples/cap18/remoto motor          # terminal 1
go run ./examples/cap18/remoto worker w1      # terminal 2 — depois, kill -9
go run ./examples/cap18/remoto worker w2      # terminal 3
GRPC_GO_LOG_SEVERITY_LEVEL=info go test -tags defeito -v ./examples/cap18/keepalive/
go test -run '^$' -bench . -cpu 1,8 ./examples/cap18/custo/
```

O contrato está em `internal/transport/grpc/proto`; o código gerado, em
`internal/transport/grpc/gen` (`make ferramentas-proto proto`).
