# Capítulo 13 — Property-based e fuzzing

| Diretório | O que mostra |
|---|---|
| `numero/` | o enigma: a forma canônica que decodifica para `any` e perde o último dígito de inteiros acima de 2^53 (`-tags defeito`). O corpus em `testdata/fuzz` guarda a entrada que o fuzzer encontrou: `go test -tags defeito ./examples/cap13/numero/` falha nela sem fuzzing nenhum |

```bash
go test -tags defeito -run '^$' -fuzz FuzzPreservaInteiros ./examples/cap13/numero/
```

As propriedades da máquina de estados e os alvos de fuzzing são código do Enxame:
`internal/core/job/property_test.go` e `internal/store/serde/serde_fuzz_test.go`.
