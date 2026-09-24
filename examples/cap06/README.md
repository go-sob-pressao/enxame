# Capítulo 6 — Padrões de concorrência

| Diretório | O que mostra |
|---|---|
| `pipeline/` | três estágios canceláveis, com propriedade de dado clara |
| `semaforo/` | semáforo com channel e com `golang.org/x/sync/semaphore` |
| `ctxsombreado/` | o enigma: o `errgroup` que não cancelava (`-tags defeito`) |

O pool do M1 é código do Enxame: `internal/worker/pool.go`.
