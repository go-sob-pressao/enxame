# Reverter uma versão

**Situação:** uma versão nova do `enxamed` se comporta mal.

**Impacto:** o que a versão nova quebrou. Reverter é seguro para o
trabalho em curso: é a mesma atualização gradual, no sentido
inverso (Teste de Realidade #4).

## Agir

```sh
kubectl rollout undo statefulset/enxame
kubectl rollout status statefulset/enxame
```

Um nó por vez, do maior para o menor; cada um só sai quando o
anterior está pronto.

## Não fazer

- Não reverta se a versão nova aplicou uma migração que a antiga não
  entende. As migrações do Enxame são do tipo expandir e depois
  contrair (Cap. 17): uma versão só acrescenta ou alarga, e o que a
  anterior usava sai numa versão seguinte. Confira nas notas da
  versão antes.
- Não apague os pods todos de uma vez para "acelerar".

## Verificar

`kubectl rollout status` concluído, os SLOs de volta, e
`sum(enxame_particoes)` em 512.
