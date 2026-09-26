package policy_test

import (
	"testing"
	"time"

	"github.com/go-sob-pressao/enxame/internal/core/policy"
)

func TestTokenBucket(t *testing.T) {
	b := &policy.TokenBucket{Taxa: 10, Rajada: 5}
	t0 := time.Date(2026, 9, 26, 14, 0, 0, 0, time.UTC)
	for i := range 5 { // a rajada inteira de uma vez
		if ok, _ := b.Tomar(t0); !ok {
			t.Fatalf("ficha %d negada", i+1)
		}
	}
	ok, espera := b.Tomar(t0)
	if ok || espera != 100*time.Millisecond {
		t.Fatalf("sexta: %v, espera %v", ok, espera)
	}
	if ok, _ := b.Tomar(t0.Add(100 * time.Millisecond)); !ok {
		t.Fatal("depois de 100 ms deveria haver uma ficha")
	}
	// Em um segundo, dez fichas a 10/s; o balde não passa de 5.
	aceitas := 0
	for ms := 0; ms <= 1000; ms += 10 {
		if ok, _ := b.Tomar(t0.Add(time.Second +
			time.Duration(ms)*time.Millisecond)); ok {
			aceitas++
		}
	}
	if aceitas < 14 || aceitas > 16 { // 5 da rajada + ~10 da taxa
		t.Fatalf("%d fichas em um segundo e pouco", aceitas)
	}
	// O relógio que volta não cria fichas.
	b2 := &policy.TokenBucket{Taxa: 1, Rajada: 1}
	b2.Tomar(t0)
	if ok, _ := b2.Tomar(t0.Add(-time.Hour)); ok {
		t.Fatal("o relógio voltou e criou ficha")
	}
}
