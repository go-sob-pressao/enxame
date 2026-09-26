package portas_test

import (
	"net/http"
	"testing"

	"github.com/go-sob-pressao/enxame/examples/cap20/portas"
)

func entregas(
	t *testing.T,
	lento bool,
	n int,
	f func(*http.Client, string) (int, error),
) int64 {
	t.Helper()
	e := portas.NovoEndpoint(lento)
	defer e.Close()
	c := &http.Client{Transport: &http.Transport{}}
	for range n {
		if s, err := f(c, e.URL); err != nil || s != 200 {
			t.Fatalf("%d %v", s, err)
		}
	}
	return e.Conexoes.Load()
}

// livro:inicio portas-teste

// Duzentas entregas ao mesmo endpoint, uma depois da outra.
func TestConexoesPorEntrega(t *testing.T) {
	semFechar := entregas(t, false, 200, portas.EntregarSemFechar)
	ingenuo := entregas(t, false, 200, portas.EntregarIngenuo)
	correto := entregas(t, false, 200, portas.Entregar)
	t.Logf("sem fechar: %d conexões; fechando sem ler: %d; "+
		"lendo e fechando: %d", semFechar, ingenuo, correto)
	// O endpoint lento: 20 entregas, cada uma leva ~100 ms.
	ingenuo = entregas(t, true, 20, portas.EntregarIngenuo)
	correto = entregas(t, true, 20, portas.Entregar)
	t.Logf("endpoint lento, 20 entregas — fechando sem ler: %d; "+
		"lendo e fechando: %d", ingenuo, correto)
	if correto != 1 {
		t.Fatalf("lendo o corpo, %d conexões", correto)
	}
}

// livro:fim portas-teste
