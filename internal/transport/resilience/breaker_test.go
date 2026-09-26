package resilience_test

import (
	"errors"
	"testing"
	"time"

	"github.com/go-sob-pressao/enxame/internal/transport/resilience"
)

func TestBreaker(t *testing.T) {
	b := &resilience.Breakers{Limiar: 3, Pausa: time.Minute}
	t0 := time.Date(2026, 9, 26, 14, 0, 0, 0, time.UTC)
	passo := func(agora time.Time, ok bool) {
		t.Helper()
		if err := b.Permitir("e1", agora); err != nil {
			t.Fatalf("%v: recusou: %v", agora.Sub(t0), err)
		}
		b.Registrar("e1", agora, ok)
	}
	for i := range 3 { // três falhas seguidas
		passo(t0.Add(time.Duration(i)*time.Second), false)
	}
	if err := b.Permitir("e1", t0.Add(30*time.Second)); !errors.Is(err,
		resilience.ErrAberto) {
		t.Fatalf("aberto deveria recusar: %v", err)
	}
	if err := b.Permitir("e2", t0); err != nil {
		t.Fatalf("outro endpoint afetado: %v", err)
	}
	// A pausa acabou: uma chamada de teste passa, a segunda não.
	depois := t0.Add(2 * time.Minute)
	if err := b.Permitir("e1", depois); err != nil {
		t.Fatalf("meio-aberto deveria deixar o teste: %v", err)
	}
	if err := b.Permitir("e1", depois); !errors.Is(err,
		resilience.ErrAberto) {
		t.Fatal("meio-aberto deixou passar duas")
	}
	b.Registrar("e1", depois, false) // o teste falhou: abre de novo
	if s := b.Estado("e1", depois.Add(30*time.Second)); s !=
		resilience.Aberto {
		t.Fatalf("estado %v", s)
	}
	passo(depois.Add(2*time.Minute), true) // o teste passou: fecha
	if s := b.Estado("e1", depois.Add(2*time.Minute)); s !=
		resilience.Fechado {
		t.Fatalf("estado %v", s)
	}
	// Uma falha e um sucesso não abrem: as falhas são seguidas.
	passo(depois.Add(3*time.Minute), false)
	passo(depois.Add(3*time.Minute), true)
	passo(depois.Add(3*time.Minute), false)
	passo(depois.Add(3*time.Minute), false)
	if s := b.Estado("e1", depois.Add(3*time.Minute)); s !=
		resilience.Fechado {
		t.Fatalf("estado %v", s)
	}
}
