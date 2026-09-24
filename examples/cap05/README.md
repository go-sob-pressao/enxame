# Capítulo 5 — `select` e o `context`

| Diretório | O que mostra |
|---|---|
| `buscaativa/` | `select` com `default` num laço: a espera que queima CPU |
| `causa/` | `context.Canceled` × `context.DeadlineExceeded` e `context.Cause` |
| `assincrono/` | o enigma: trabalho assíncrono que herda o contexto da requisição (`-tags defeito`) e `context.WithoutCancel` |

O long-poll de três vias, estudo de caso do capítulo, é código do Enxame:
`internal/queue/longpoll.go`.
