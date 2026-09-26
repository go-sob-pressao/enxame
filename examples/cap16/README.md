# Capítulo 16 — Idempotência na prática

| Diretório | O que mostra |
|---|---|
| `gateway/` | um gateway de pagamento falso que aceita chave de idempotência e sabe "perder" respostas depois de debitar |
| `chave/` | o enigma: a chave gerada com `uuid.NewV7()` a cada tentativa (`-tags defeito`: cinco tentativas, cinco débitos); o Experimento 16.1, com `job.IdempotencyKey`: cinco tentativas, um débito |
| `dedup/` | a tabela de deduplicação própria, para quando o sistema externo não aceita chave; e a versão local, com a chave e o efeito na mesma transação. Precisa de `ENXAME_DB_DSN` |
| `jitter/` | mil clientes tentando de novo depois da mesma falha, com e sem jitter: os dados da Figura 16.2 |

```bash
go test -tags defeito ./examples/cap16/chave/
go test -v -run CincoTentativas ./examples/cap16/chave/
go run ./examples/cap16/jitter > jitter.csv
```

As chaves derivadas estão em `internal/core/id/idempotencia.go`; a política
de retry, em `internal/core/policy`; os agendamentos, em `internal/engine/cron`.
