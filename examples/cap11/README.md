# Capítulo 11 — Testando concorrência

| Diretório | O que mostra |
|---|---|
| `intermitente/` | o teste que espera com `time.Sleep` e falha de vez em quando (`-tags defeito`, rode com `-count=2000`); a versão com `synctest.Wait` |
| `trava/` | o enigma: o teste com `synctest` que nunca termina — esperar um `Mutex` não é bloqueio durável (`-tags defeito`, rode com `-timeout 10s`) |

O relógio injetável e o timer pump são código do Enxame:
`internal/simulation/clock/` e `internal/engine/timer/`.
