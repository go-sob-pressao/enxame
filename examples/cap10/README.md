# Capítulo 10 — TDD em Go, sem dogma

| Diretório | O que mostra |
|---|---|
| `ordem/` | o enigma: testes que compartilham uma fixture de pacote e só passam na ordem do arquivo (`-tags defeito`, com `-shuffle=on` ou `-count=2`); a versão correta, com fixture por teste |

A especificação da máquina de estados, escrita por TDD, é código do Enxame:
`internal/core/job/spec_test.go` e `internal/core/job/rules.go`. O ciclo
vermelho → verde → refatorar está nos três commits que levam à tag
`cap-10-tdd`:

```bash
git log --oneline cap-09..cap-10-tdd
```
