# Capítulo 9 — O `race detector` como professor

| Diretório | O que mostra |
|---|---|
| `caminho/` | o enigma: a corrida que só existe no caminho de retry; a CI com `-race` passa porque nenhum teste passa por ele (`-tags defeito`) |
| `errignorado/` | `err` ignorado com `_` (Anti-Pattern #11): o commit que falhou em silêncio |

As três corridas reais do Enxame estão na tag intermediária `cap-09-corridas`
e corrigidas na `cap-09`:

```bash
git checkout cap-09-corridas && go test -race ./internal/...   # mais de cem relatórios; o número varia
git diff cap-09-corridas cap-09 -- internal/                    # as correções
```
