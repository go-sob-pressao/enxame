package avalanche_test

import (
	"math/rand/v2"
	"testing"
	"time"

	avalanche "github.com/go-sob-pressao/enxame/missoes/04-avalanche"
)

// livro:inicio missao-04-teste

// 2.000 entregas, o endpoint fora do ar por uma hora; quando volta,
// aguenta 100 requisições por segundo. Todas precisam chegar em até
// duas horas, sem derrubar o endpoint de novo.
func TestMissao(t *testing.T) {
	sorte := rand.New(rand.NewPCG(4, 4)).Float64
	r := avalanche.Simular(avalanche.Config{
		Entregas: 2000, Queda: time.Hour, Capacidade: 100,
		Limite: 2 * time.Hour,
	}, avalanche.ProximaTentativa, sorte)
	t.Logf("entregues %d de 2000; pico %d/s; recaídas %d; "+
		"última em %v", r.Entregues, r.Pico, r.Recaidas, r.UltimaEm)
	if r.Entregues != 2000 || r.Recaidas != 0 {
		t.Fatal("a avalanche venceu")
	}
}

// livro:fim missao-04-teste
