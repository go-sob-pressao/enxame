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
