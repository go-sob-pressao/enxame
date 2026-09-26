# Capítulo 23 — Quem está vivo?

| Diretório | O que mostra |
|---|---|
| `detector/` | um dia de batidas simulado: timeout fixo × phi accrual, com pausas de GC (cenário 1) e com um nó rápido e um lento (cenário 2) |
| `gossip/` | rodadas até uma novidade chegar a todos, em push e em push-pull, de 8 a 4096 nós |

```bash
go run ./examples/cap23/detector
go run ./examples/cap23/gossip
go run ./examples/cap23/gossip -curva 1024
```

O detector está em `internal/cluster/membership/detector.go`; o
membership pelo banco, em `membro.go`; a suíte de conformidade, em
`internal/cluster/coordinator/conformidade.go`.

Três nós locais para os Experimentos 23.1 e 23.2:

```bash
make up
for i in 1 2 3; do
  go run ./cmd/enxamed -no no-$i -http :808$i -grpc :723$i \
      -tokens t1:loja -worker-token w &
done
ENXAME_API=http://localhost:8081 ENXAME_TOKEN=t1 \
    go run ./cmd/enxamectl cluster members
kill -STOP <pid do no-3>; sleep 3; kill -CONT <pid do no-3>   # 23.1
kill -9 <pid do no-3>                                          # 23.2
```
