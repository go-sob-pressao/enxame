package membership_test

import (
	"testing"
	"time"

	"github.com/go-sob-pressao/enxame/internal/cluster/membership"
)

var t0 = time.Date(2026, 9, 26, 12, 0, 0, 0, time.UTC)

// batidas registra n batidas a cada intervalo e devolve a última.
func batidas(d membership.Detector, n int, intervalo time.Duration) time.Time {
	em := t0
	for range n {
		d.Batida(em)
		em = em.Add(intervalo)
	}
	return em.Add(-intervalo)
}

func TestFixoSemMeioTermo(t *testing.T) {
	f := &membership.Fixo{Limite: 2 * time.Second}
	ultima := batidas(f, 10, 500*time.Millisecond)
	if e := f.Estado(ultima.Add(2 * time.Second)); e != membership.Vivo {
		t.Fatalf("no limite: %v", e)
	}
	if e := f.Estado(ultima.Add(2001 * time.Millisecond)); e !=
		membership.Morto {
		t.Fatalf("depois do limite: %v", e)
	}
}

// Com batidas regulares a cada 500 ms, o phi cresce com o silêncio e
// passa por suspeito antes de morto.
func TestPhiPassaPorSuspeita(t *testing.T) {
	p := membership.NovoPhi(500 * time.Millisecond)
	ultima := batidas(p, 50, 500*time.Millisecond)
	var visto []membership.Estado
	for ms := 0; ms <= 8000; ms += 100 {
		e := p.Estado(ultima.Add(time.Duration(ms) * time.Millisecond))
		if len(visto) == 0 || visto[len(visto)-1] != e {
			visto = append(visto, e)
		}
	}
	want := []membership.Estado{membership.Vivo, membership.Suspeito,
		membership.Morto}
	if len(visto) != 3 || visto[0] != want[0] || visto[1] != want[1] ||
		visto[2] != want[2] {
		t.Fatalf("estados: %v", visto)
	}
}

// Um nó com pausas longas na história tolera uma pausa nova do mesmo
// tamanho; um nó regular, não.
func TestPhiAprendeComAHistoria(t *testing.T) {
	regular := membership.NovoPhi(500 * time.Millisecond)
	ultimaR := batidas(regular, 100, 500*time.Millisecond)
	pausas := membership.NovoPhi(500 * time.Millisecond)
	em := t0
	for i := range 100 {
		pausas.Batida(em)
		if i%10 == 9 {
			em = em.Add(3 * time.Second) // uma pausa a cada dez
		} else {
			em = em.Add(500 * time.Millisecond)
		}
	}
	ultimaP := em.Add(-500 * time.Millisecond)
	silencio := 5 * time.Second
	eR, eP := regular.Estado(ultimaR.Add(silencio)),
		pausas.Estado(ultimaP.Add(silencio))
	t.Logf("5 s de silêncio: regular phi %.1f (%v); com pausas phi %.1f (%v)",
		regular.Valor(ultimaR.Add(silencio)), eR,
		pausas.Valor(ultimaP.Add(silencio)), eP)
	if eR != membership.Morto || eP == membership.Morto {
		t.Fatalf("regular %v, com pausas %v", eR, eP)
	}
}

// Quem bate uma vez só e some também morre: antes de haver história, o
// detector supõe o intervalo esperado.
func TestPhiSemHistoria(t *testing.T) {
	p := membership.NovoPhi(500 * time.Millisecond)
	if e := p.Estado(t0); e != membership.Vivo {
		t.Fatalf("antes da primeira batida: %v", e)
	}
	p.Batida(t0)
	if e := p.Estado(t0.Add(10 * time.Second)); e != membership.Morto {
		t.Fatalf("uma batida e 10 s de silêncio: %v", e)
	}
}
