# Endpoint de webhook desativado

**Situação:** um cliente diz que parou de receber webhooks.

**Impacto:** as mensagens para aquele endpoint não são entregues;
nenhuma se perde, e todas saem quando ele for reativado.

## Confirmar

```sh
psql "$ENXAME_DB_DSN" -c "
  SELECT endpoint_id, url, disabled_at, disabled_reason
  FROM webhook_endpoint WHERE namespace = '<namespace>'"
```

O estado do breaker, no dono da partição:
`GET /v1/webhooks/endpoints/{id}/state`.

## Agir

1. **`disabled_at` preenchido:** o endpoint foi desativado por falhas
   persistentes. A última tentativa (`webhook_attempt`) diz o status
   que o cliente respondia. Com o cliente de volta, reinscreva o
   endpoint.
2. **Breaker aberto, sem desativação:** o endpoint falha agora; o
   breaker volta a tentar sozinho depois da pausa.

## Não fazer

- Não reenvie mensagens à mão por fora do Enxame: o cliente as
  receberia duas vezes, sem a mesma chave.
