package json_test

import (
	v1 "encoding/json"
	v2 "encoding/json/v2"
	"testing"
	"time"

	ex "github.com/go-sob-pressao/enxame/examples/cap19/json"
)

// livro:inicio v1-v2

// A mesma entrada, nos dois pacotes. O teste não afirma que um está
// certo: registra o que cada um faz.
func TestV1xV2(t *testing.T) {
	casos := []struct{ nome, entrada string }{
		{"chave em outra caixa", `{"CLIENTE":"ana"}`},
		{"chave repetida", `{"cliente":"ana","cliente":"bia"}`},
		{"UTF-8 inválido", "{\"cliente\":\"a\xffb\"}"},
		{"número grande num int64", `{"valor":9007199254740993}`},
		{"campo desconhecido", `{"clientes":"ana"}`},
	}
	for _, c := range casos {
		var p1, p2 ex.Pedido
		e1 := v1.Unmarshal([]byte(c.entrada), &p1)
		e2 := v2.Unmarshal([]byte(c.entrada), &p2)
		t.Logf("%-24s v1: %q %d %v", c.nome, p1.Cliente, p1.Valor, e1)
		t.Logf("%-24s v2: %q %d %v", "", p2.Cliente, p2.Valor, e2)
	}
	var zero ex.Pedido
	s1, _ := v1.Marshal(zero)
	s2, _ := v2.Marshal(zero)
	t.Logf("zero em v1: %s", s1)
	t.Logf("zero em v2: %s", s2)
}

// livro:fim v1-v2

// Ausente e zero são coisas diferentes, e só o ponteiro as distingue.
func TestAusenteXZero(t *testing.T) {
	var a, b ex.Pedido
	_ = v2.Unmarshal([]byte(`{}`), &a)
	_ = v2.Unmarshal([]byte(`{"parcelas":0}`), &b)
	if a.Parcelas != nil || b.Parcelas == nil || *b.Parcelas != 0 {
		t.Fatalf("ausente %v, zero %v", a.Parcelas, b.Parcelas)
	}
	// omitempty: v1 omite o 0 de desconto; v2 só omite o que é vazio
	// em JSON — "", null, [], {} —, e o 0 aparece.
	p := ex.Pedido{CriadoEm: time.Unix(0, 0).UTC()}
	s1, _ := v1.Marshal(p)
	s2, _ := v2.Marshal(p)
	t.Logf("omitempty v1: %s", s1)
	t.Logf("omitempty v2: %s", s2)
}

// livro:inicio custo-json

func BenchmarkCodificarV1(b *testing.B) {
	r := resposta()
	for b.Loop() {
		if _, err := v1.Marshal(r); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkCodificarV2(b *testing.B) {
	r := resposta()
	for b.Loop() {
		if _, err := v2.Marshal(r); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkDecodificarV1(b *testing.B) {
	dados, _ := v1.Marshal(resposta())
	for b.Loop() {
		var r ex.Resposta
		if err := v1.Unmarshal(dados, &r); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkDecodificarV2(b *testing.B) {
	dados, _ := v1.Marshal(resposta())
	for b.Loop() {
		var r ex.Resposta
		if err := v2.Unmarshal(dados, &r); err != nil {
			b.Fatal(err)
		}
	}
}

// livro:fim custo-json

func resposta() ex.Resposta {
	return ex.Resposta{ID: "0199a3b4-0000-7000-8000-000000000001",
		Queue: "default", Kind: "email.enviar", State: "running",
		Attempt: 2, MaxAttempts: 25,
		ScheduledAt: time.Date(2026, 9, 26, 14, 0, 0, 0, time.UTC),
		Tags:        []string{"cliente:42", "urgente"}}
}
