# Líder oscilando

**Alarme:** `LiderOscilando` — o mapa de partições mudou mais de 5
vezes em 10 minutos.

**Impacto:** cada mapa novo move partições, e cada movimento drena as
tentativas em curso. O trabalho anda, aos solavancos; o atraso sobe.

## Confirmar

```sh
psql "$ENXAME_DB_DSN" -c "
  SELECT epoch, term, created_at FROM coord_assignment
  ORDER BY epoch DESC LIMIT 10"
psql "$ENXAME_DB_DSN" -c "
  SELECT node, now() - beat_at AS silencio,
         now() - started_at AS no_ar
  FROM cluster_member ORDER BY node"
```

`term` mudando é o líder mudando; `term` fixo com `epoch` subindo é o
mesmo líder refazendo o mapa, porque um membro entra e sai.

## Agir

1. **Um nó com `no_ar` de segundos, repetidamente:** ele está
   reiniciando. `kubectl get pods` mostra os `RESTARTS`, e
   `kubectl describe pod` o motivo: `OOMKilled` (memória: reveja o
   limite e o `GOMEMLIMIT`), ou `Liveness probe failed` (a probe não
   pode depender do banco — Cap. 32).
2. **Silêncios longos sem reinício:** o nó não consegue bater — pausa
   de coleta de lixo, CPU estrangulada, rede. O perfil de CPU (`-diag`)
   e a métrica de throttling do contêiner respondem.

## Não fazer

- Não aumente o lease para "estabilizar": o mapa fica mais lento para
  reagir à próxima queda de verdade.

## Verificar

`epoch` parado por 10 minutos.

## Escalar

O dono do sistema (`docs/dono.md`).
