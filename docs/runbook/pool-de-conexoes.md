# Pool de conexões esgotado

**Alarme:** `PoolDeConexoesEsgotado` — um nó esperando conexão do
banco mais de uma vez por segundo, por 10 minutos.

**Impacto:** cada espera soma latência a uma requisição ou a uma
reserva de job. Sustentado, vira fila.

## Confirmar

Painel "Enxame", linha Banco: conexões em uso no máximo do pool? O
p99 das rotas subiu junto?

```sh
psql "$ENXAME_DB_DSN" -c "
  SELECT state, wait_event_type, count(*) FROM pg_stat_activity
  WHERE datname = current_database() GROUP BY 1, 2"
```

## Agir

1. **Conexões ocupadas com consultas lentas** (`active` alto): o
   banco é o gargalo; aumentar o pool só aumenta a fila no banco.
2. **Conexões presas em transação ociosa** (`idle in transaction`):
   algum código segura a transação; o perfil de goroutines (`-diag`)
   mostra quem.
3. **Pouca conexão para a carga:** aumente o pool no DSN
   (`pool_max_conns`), respeitando o `max_connections` do banco
   dividido pelos nós.

## Verificar

`enxame_db_esperas_total` parando de crescer.

## Escalar

O dono do sistema; o DBA, se o banco estiver saturado.
