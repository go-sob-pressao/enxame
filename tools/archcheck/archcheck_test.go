package main

import (
	"os"
	"slices"
	"strings"
	"testing"
)

// livro:inicio teste-loja

// testdata/loja é um módulo com uma violação plantada para cada
// regra. A lista esperada fica em testdata/loja.esperado: uma regra
// nova ganha uma linha ali, e a mudança aparece no diff do pull
// request.
func TestVerificarModuloComViolacoesPlantadas(t *testing.T) {
	vs, err := verificar("testdata/loja")
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, v := range vs {
		got = append(got, v.String())
	}
	dados, err := os.ReadFile("testdata/loja.esperado")
	if err != nil {
		t.Fatal(err)
	}
	want := strings.Split(strings.TrimSpace(string(dados)), "\n")
	if !slices.Equal(got, want) {
		t.Errorf(
			"violações divergentes\n got:\n  %s\nwant:\n  %s",
			join(got),
			join(want),
		)
	}
}

// livro:fim teste-loja

// A saída precisa ser idêntica entre execuções: é a mesma exigência que
// o livro faz à simulação determinística.
func TestVerificarEDeterministico(t *testing.T) {
	primeira, err := verificar("testdata/loja")
	if err != nil {
		t.Fatal(err)
	}
	for range 5 {
		outra, err := verificar("testdata/loja")
		if err != nil {
			t.Fatal(err)
		}
		if !slices.Equal(primeira, outra) {
			t.Fatalf(
				"saída mudou entre execuções:\n%v\n%v",
				primeira,
				outra,
			)
		}
	}
}

func TestDentroRespeitaFronteiraDeSegmento(t *testing.T) {
	casos := []struct {
		caminho, prefixo string
		want             bool
	}{
		{"internal/core", "internal/core", true},
		{"internal/core/job", "internal/core", true},
		{"internal/corex", "internal/core", false},
		{"internal/cluster/raftcoord", "internal/cluster/raft", false},
		{"net/http", "net", true},
		{"netip", "net", false},
		{"pkg/enxame", "pkg", true},
	}
	for _, c := range casos {
		if got := dentro(c.caminho, c.prefixo); got != c.want {
			t.Errorf(
				"dentro(%q, %q) = %v, want %v",
				c.caminho,
				c.prefixo,
				got,
				c.want,
			)
		}
	}
}

func TestRelativo(t *testing.T) {
	if got := relativo("exemplo.com/loja/internal/core", "exemplo.com/loja"); got != "internal/core" {
		t.Errorf("got %q", got)
	}
	if got := relativo("exemplo.com/loja", "exemplo.com/loja"); got != "." {
		t.Errorf("got %q", got)
	}
}

func join(xs []string) string { return strings.Join(xs, "\n  ") }

// Um módulo que não compila não pode ser declarado "arquitetura ok".
func TestVerificarRecusaModuloQueNaoCompila(t *testing.T) {
	if _, err := verificar("testdata/quebrado"); err == nil {
		t.Fatal("esperava erro de compilação, veio nil")
	}
}
