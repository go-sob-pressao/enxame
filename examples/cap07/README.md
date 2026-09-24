# Capítulo 7 — Vazamento de goroutine

| Diretório | O que mostra |
|---|---|
| `vazamentos/` | as quatro formas clássicas de vazar, cada uma com o teste que a flagra (goleak) e a correção |
| `perfil/` | o perfil `goroutineleak` (Go 1.27) apontando o vazamento num serviço em execução |

A correção do handler do enigma #1 (Capítulo 1) está em `examples/cap01/handler/correto.go`.
