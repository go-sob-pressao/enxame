# Capítulo 22 — O que a rede quebra

| Diretório | O que mostra |
|---|---|
| `foradeordem/` | o incidente: dois nós entregam o `pedido.cancelado` antes do `pedido.criado`; um nó, não — até a primeira falha passageira |
| `relogio/` | a leitura monotônica de um `time.Time` e o que a serialização remove |
| `lease/` | o enigma: o lease calculado com o relógio de um nó e comparado com o de outro |

```bash
make up
export ENXAME_DB_DSN=postgres://postgres:enxame@localhost:5432/enxame?sslmode=disable
go run ./examples/cap22/foradeordem -nos 1
go run ./examples/cap22/foradeordem -nos 2
go run ./examples/cap22/foradeordem -nos 1 -falhas 10
go run ./examples/cap22/relogio
go test -v ./examples/cap22/lease/
```

O contrato do plano de controle que a Parte V vai implementar está em
`internal/cluster/coordinator/coordinator.go`.
