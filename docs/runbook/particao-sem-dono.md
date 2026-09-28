# Partição sem dono

**Alarme:** `ParticaoSemDono` — menos de 512 partições com posse
válida por 2 minutos.

**Impacto:** os jobs, workflows e webhooks das partições sem dono não
andam. Nada se perde: o trabalho espera no banco. A API continua
aceitando.

## Confirmar

Quantas estão sem dono, e quem o mapa diz que deveria tê-las:

```sh
psql "$ENXAME_DB_DSN" -c "
  SELECT count(*) FILTER (WHERE owner IS NULL
           OR lease_expires_at < now()) AS sem_dono
  FROM partition_lease"
psql "$ENXAME_DB_DSN" -c "
  SELECT dono, count(*) FROM coord_assignment a,
         jsonb_array_elements_text(a.owners) dono
  WHERE a.epoch = (SELECT max(epoch) FROM coord_assignment)
  GROUP BY dono"
```

O mapa aponta para um nó que não bate há mais de 10 s?

```sh
psql "$ENXAME_DB_DSN" -c "
  SELECT node, now() - beat_at AS silencio
  FROM cluster_member ORDER BY node"
kubectl get pods -l app.kubernetes.io/name=enxamed
```

## Agir

1. **Um nó sumiu e o mapa ainda aponta para ele.** O líder refaz o
   mapa quando o detector o declara morto, em segundos. Se passaram
   2 minutos, confira o líder (`lider-oscilando.md`).
2. **O nó está vivo, mas não adquire.** Veja o que ele acha de si:

   ```sh
   kubectl get --raw \
     "/api/v1/namespaces/$NS/pods/enxame-0:8080/proxy/statusz"
   ```

   `"particoes": 0` com `"banco": "ok"` é um nó sem posse: os avisos
   dele (`kubectl logs enxame-0 | grep WARN`) dizem por quê.
   `"banco"` com erro é um nó que não chega ao banco: veja a rede
   dele, não o banco.
3. **Um pod em `Pending`.** Sem nó para ele (a regra de espalhamento
   pede um nó por pod): libere um nó ou acrescente outro.

## Não fazer

- Não apague nem edite linhas de `partition_lease`: o `range_id` é a
  cerca que impede dois donos (Cap. 25).
- Não reinicie todos os nós de uma vez. Um de cada vez:
  `kubectl delete pod enxame-N`, esperando o anterior ficar pronto.

## Verificar

`sem_dono` em zero, e a fila das partições voltando a andar
(`enxame_fila_mais_antigo_segundos` caindo).

## Escalar

Depois de 15 minutos sem causa: o dono do sistema (`docs/dono.md`).
