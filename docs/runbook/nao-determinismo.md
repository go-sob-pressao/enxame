# Run parado com NonDeterministicError

**Situação:** um workflow parou, e o erro da última tentativa é
`NonDeterministicError`.

**Impacto:** o run não avança nem termina. Os passos já gravados
continuam gravados.

## Confirmar

```sh
psql "$ENXAME_DB_DSN" -c "
  SELECT run_id, workflow_type, code_version, next_step_seq
  FROM workflow_run WHERE state = 'running'
  AND run_id IN (SELECT run_id FROM job
                 WHERE state = 'discarded' AND run_id IS NOT NULL)"
```

## Agir

1. O código do workflow mudou a ordem ou o nome dos passos sem
   versionar (Cap. 17). Volte a versão anterior do worker, ou
   publique a mudança atrás de `workflow.Version`.
2. Com o código compatível, o run segue na próxima tentativa.

## Não fazer

- Não edite `workflow_step`: o histórico é o que torna o replay
  seguro.
