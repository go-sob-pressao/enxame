package serde_test

import (
	"encoding/json"
	"testing"

	"github.com/go-sob-pressao/enxame/internal/store/serde"
)

// livro:inicio fuzz-serde

// FuzzCanonicalPreservaInteiros: um inteiro de até 19 dígitos, dentro
// de um objeto, sobrevive à forma canônica, dígito por dígito. O fuzzer
// muta o texto do número — como ele chega numa requisição —, e não um
// int64: mutando int64, o fuzzer quase não gera valores acima de 2^53.
func FuzzCanonicalPreservaInteiros(f *testing.F) {
	f.Add("42")
	f.Fuzz(func(t *testing.T, digitos string) {
		if !inteiro(digitos) {
			return
		}
		entrada := `{"pedido":` + digitos + `}`
		got, err := serde.Canonical([]byte(entrada))
		if err != nil {
			t.Fatal(err)
		}
		if string(got) != entrada {
			t.Fatalf("%s virou %s", entrada, got)
		}
	})
}

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

// FuzzCanonicalIdempotente: a forma canônica de uma forma canônica é
// ela mesma, para qualquer JSON válido.
func FuzzCanonicalIdempotente(f *testing.F) {
	f.Add([]byte(`{"b":[1,2.5,"x"],"a":null}`))
	f.Fuzz(func(t *testing.T, data []byte) {
		if !json.Valid(data) {
			return
		}
		um, err := serde.Canonical(data)
		if err != nil {
			return // um valor com dados depois dele, por exemplo
		}
		dois, err := serde.Canonical(um)
		if err != nil || string(um) != string(dois) {
			t.Fatalf("%q → %q → %q (%v)", data, um, dois, err)
		}
	})
}

// livro:fim fuzz-serde
