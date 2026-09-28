# Runbooks

Um por alarme, e um para cada procedimento que alguém vai precisar
fazer com pressa (Capítulo 32). Cada um responde, nesta ordem: o que
disparou, qual o impacto, como confirmar, o que fazer, o que não
fazer, como saber que passou e quem chamar.

| Alarme ou situação | Runbook |
|---|---|
| `ParticaoSemDono` | `particao-sem-dono.md` |
| `LiderOscilando` | `lider-oscilando.md` |
| `FilaEnvelhecendo`, `AtrasoDeAgendamento` | `fila-crescendo.md` |
| `PoolDeConexoesEsgotado` | `pool-de-conexoes.md` |
| `OrcamentoQueimandoRapido`, `OrcamentoQueimandoDevagar` | `orcamento-de-erro.md` |
| cliente reclama de webhook que parou | `endpoint-desativado.md` |
| run parado com `NonDeterministicError` | `nao-determinismo.md` |
| versão nova com defeito | `rollback-versao.md` |

Os comandos supõem o Enxame instalado pelo `deploy/k8s` no namespace
corrente e `ENXAME_DB_DSN` com o DSN do banco. O dono do sistema, o
plantão e a escala estão em `docs/dono.md`.

A imagem do `enxamed` não tem shell. Para olhar de dentro de um pod —
processos, rede, o `/statusz` pelo `localhost` —, um contêiner de
depuração efêmero, com o mesmo usuário sem privilégios do pod:

```sh
kubectl debug enxame-0 -it --image=busybox:1.37 --target=enxamed \
  --profile=restricted --custom=deploy/k8s/depurar.json -- sh
```

Sem o `--custom`, o busybox rodaria como root, e o pod — que exige
`runAsNonRoot` — o recusaria.
