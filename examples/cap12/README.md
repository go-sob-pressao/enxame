# Capítulo 12 — Dublês e fronteiras

| Diretório | O que mostra |
|---|---|
| `ingenuo/` | o enigma: o fake de Store que aceitava duas `unique_key` iguais, posto diante da suíte de contrato (`-tags defeito`) |

O Store e a suíte de contrato são código do Enxame:
`internal/engine/store.go` (a interface, no consumidor),
`internal/store/storetest` (a suíte) e `internal/store/{memory,sqlite,postgres}`.
A suíte contra o Postgres roda com `make up` e `make integration`.
