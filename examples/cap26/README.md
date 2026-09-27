# Capítulo 26 — Dividir o trabalho

| Diretório | O que mostra |
|---|---|
| `cluster/` | as peças do Teste de Realidade #3: worker remoto que grava o efeito de cada job, gerador de carga pela API, conferência |

```bash
make up
go test -tags integration -run Realidade3 -v ./test/integration/
go run ./examples/cap22/foradeordem -nos 2        # o incidente do Cap. 22, agora em ordem
go test -v ./internal/cluster/routing/             # o enigma #25
```

A função de partição está em `internal/core/id/particao.go`; a ordem
por chave, em `internal/store/postgres/busca.go` (`cabeca`); o
rebalanceamento, em `internal/cluster/sharding`; o roteamento, em
`internal/cluster/routing`.
