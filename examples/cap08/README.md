# Capítulo 8 — Memória compartilhada e o memory model

| Diretório | O que mostra |
|---|---|
| `checar/` | o enigma: "verificar e depois agir" com `sync.Map` — duas conexões por endpoint e o `-race` em silêncio (`-tags defeito`) |
| `flag/` | a flag `parar` sem sincronização: o memory model não garante que o laço termine (`-tags defeito -race`) |
| `copia/` | `Mutex` copiado por receptor de valor (Anti-Pattern #13) — `go vet -tags defeito` |
| `publicacao/` | publicar configuração com `atomic.Pointer` |
| `custo/` | Custo Real #2: Mutex sem e com contenção, RWMutex, atomic |
| `falsoshare/` | falso compartilhamento: dois contadores na mesma linha de cache |

O lock striped do Enxame: `internal/engine/dispatcher/stripe.go` (ADR-006).
