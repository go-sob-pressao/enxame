# Fila crescendo ou envelhecendo

**Alarmes:** `FilaEnvelhecendo` (o job mais antigo espera há mais de
5 min) e `AtrasoDeAgendamento` (p99 do atraso acima de 1 min).

**Impacto:** o trabalho da fila sai atrasado; se é uma fila de
negócio com prazo (relatórios, cobranças), o prazo está em risco.

## Confirmar

```sh
psql "$ENXAME_DB_DSN" -c "
  SELECT queue, state, count(*), min(scheduled_at) AS mais_antigo
  FROM job WHERE state IN ('available', 'running', 'retryable')
  GROUP BY 1, 2 ORDER BY 1, 2"
```

## Agir

1. **`available` crescendo, `running` estável:** falta capacidade.
   Mais workers (`kubectl scale deploy/<worker> --replicas=N`), e
   confira que cada worker tem um stream para cada `enxamed`.
2. **`running` alto e nada termina:** tentativas presas. O trace de
   uma delas (`trace_parent` do job) diz onde. O resgate as devolve
   quando o prazo sem sinal de vida vence.
3. **`retryable` crescendo:** os jobs falham. O erro está no evento
   da tentativa (`job_event`) e no trace.
4. **Um job antigo sozinho na cabeça de uma chave de ordem:** os
   seguintes da chave esperam por ele, por desenho. Resolva o job; não
   a fila.

## Não fazer

- Não cancele jobs em massa para "limpar" a fila: o trabalho é de
  alguém.

## Verificar

`enxame_fila_mais_antigo_segundos` caindo, e o alarme resolvido.

## Escalar

O dono da fila — o time da aplicação — e o dono do sistema.
