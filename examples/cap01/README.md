# Capítulo 1 — O Go que você acha que conhece

Cada diretório é um enigma, em duas versões:

| Arquivo | Build tag | O que é |
|---|---|---|
| `defeito.go` | `defeito` | o código que compila, passa em revisão e está errado |
| `correto.go` | (nenhuma) | a correção |

```bash
go run ./examples/cap01/nilinterface                 # versão correta
go run -tags defeito ./examples/cap01/nilinterface   # reproduz o enigma
go test -tags defeito ./examples/cap01/...           # os testes provam o defeito
go vet -tags defeito ./examples/cap01/mutexcopia     # o que o vet acusa
```

A versão com defeito fica fora da compilação padrão porque as ferramentas da CI
a recusariam — e é exatamente esse o ponto do capítulo.

`lacovar/` é um módulo à parte, com `go 1.21` no `go.mod`, para mostrar a
captura de variável de laço que a linguagem mudou no Go 1.22.
