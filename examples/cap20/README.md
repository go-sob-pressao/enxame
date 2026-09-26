# Capítulo 20 — Falha é o estado normal

| Diretório | O que mostra |
|---|---|
| `portas/` | o enigma: conexões TCP novas por entrega — sem fechar o corpo, fechando sem ler (com endpoint lento) e lendo até o fim |

Os experimentos do capítulo usam outros diretórios:

```bash
go run ./examples/cap16/jitter -clientes 100   # Experimento 20.1
go run ./examples/03-webhooks                  # Experimento 20.2
go test -v ./examples/cap20/portas/
```

A entrega está em `internal/delivery`; o breaker, em
`internal/transport/resilience`; a assinatura, em `internal/core/webhook`; a
verificação para quem recebe, em `pkg/webhook`.
