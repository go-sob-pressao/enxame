package numero

import (
	"testing"
)

// livro:inicio fuzz-numero

// O id do pedido chega como texto, dentro do JSON da requisição. A
// propriedade: um inteiro de até 19 dígitos sobrevive à forma canônica.
// O fuzzer muta o texto, e não um int64: é assim que o número chega.
func FuzzPreservaInteiros(f *testing.F) {
	f.Add("42")
	f.Fuzz(func(t *testing.T, digitos string) {
		if !inteiro(digitos) {
			return
		}
		entrada := `{"pedido":` + digitos + `}`
		got, err := Canonical([]byte(entrada))
		if err != nil {
			t.Fatal(err)
		}
		if string(got) != entrada {
			t.Fatalf("%s virou %s", entrada, got)
		}
	})
}

// livro:fim fuzz-numero

// inteiro aceita de 1 a 19 dígitos, sem zero à esquerda: o formato de
// um int64 positivo em JSON.
func inteiro(s string) bool {
	if len(s) == 0 || len(s) > 19 || (s[0] == '0' && len(s) > 1) {
		return false
	}
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}
