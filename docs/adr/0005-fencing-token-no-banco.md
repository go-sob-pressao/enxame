# ADR 0005 — Fencing token no banco, verificado com FOR SHARE

- **Status:** aceita
- **Capítulo:** 24
- **Data:** 2026-09-24

## Contexto

Invariante central: **em qualquer instante, no máximo um nó escreve em uma
partição**. Lease com timeout não basta: um nó congelado por GC acorda,
ainda se acha dono, e escreve.

## Opções consideradas

1. **Lease apenas por timeout.** Demonstrado no Capítulo 24: corrompe dados.
2. **Fencing token verificado por subconsulta na escrita**
   (`UPDATE … WHERE $range = (SELECT range_id FROM partition_lease …)`).
3. **Fencing token verificado com `SELECT … FOR SHARE` no início da
   transação** do dono.

## Decisão

Opção 3. Toda transação do dono começa com:

```sql
SELECT range_id FROM partition_lease WHERE partition_id = $1 FOR SHARE;
-- se range_id <> o range_id que o nó carrega: ROLLBACK e abandona a partição
```

A aquisição incrementa o token:

```sql
UPDATE partition_lease
   SET owner = $1, range_id = range_id + 1, lease_expires_at = now() + $2
 WHERE partition_id = $3 AND (owner IS NULL OR owner = $1 OR lease_expires_at < now())
RETURNING range_id;
```

## Por que a opção 2 está errada

Em `READ COMMITTED`, a verificação sem lock é um *time-of-check to
time-of-use*: o zumbi lê o `range_id` antigo, o novo dono comita a aquisição,
e a escrita do zumbi é comitada **depois** — aceita pelo banco. O `FOR SHARE`
faz a aquisição (`UPDATE`) esperar a transação do zumbi terminar; qualquer
transação seguinte do zumbi enxerga o token novo e é rejeitada.

Verificado em 2026-09-24 contra PostgreSQL 18.6, `READ COMMITTED`:

| Variante | Aquisição esperou o zumbi? | Escrita do zumbi comitada após a aquisição? |
|---|---|---|
| sem `FOR SHARE` | não | **sim** — corrupção |
| com `FOR SHARE` | sim | não |

Esse caso é o box *"O bug que parecia impossível"* do Capítulo 24: dez linhas,
revisadas, com fencing — e erradas.

## Consequências

- A correção vive no banco, não no relógio nem na aplicação (Regra 24).
- Custo: um lock compartilhado por transação de dono. Medido no Capítulo 24.
- A implementação em memória do Store (`internal/store/memory`) precisa
  reproduzir esta semântica; a suíte de contrato inclui o cenário acima.
