package timer_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"github.com/go-sob-pressao/enxame/internal/engine/timer"
	"github.com/go-sob-pressao/enxame/internal/simulation/clock"
)

// disparos registra os instantes em que o pump chamou Promote.
type disparos struct {
	mu sync.Mutex
	em []time.Time
}

func (d *disparos) promote(at time.Time) (int, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.em = append(d.em, at)
	return 1, nil
}

func (d *disparos) lista() []time.Time {
	d.mu.Lock()
	defer d.mu.Unlock()
	return append([]time.Time(nil), d.em...)
}

// livro:inicio pump-synctest

// Dentro da bolha, o pump usa o relógio real — que ali é virtual. Os
// prazos de 2 s e 5 s disparam exatamente nesses instantes, e o teste
// termina sem esperar nada de verdade.
func TestPumpDisparaNaHoraCerta(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		var d disparos
		p := timer.NewPump(clock.Real{}, d.promote)
		ctx, cancel := context.WithCancel(t.Context())
		fim := make(chan error)
		go func() { fim <- p.Run(ctx) }()

		inicio := time.Now()
		_ = p.Schedule(ctx, inicio.Add(5*time.Second))
		_ = p.Schedule(ctx, inicio.Add(2*time.Second))

		synctest.Sleep(10 * time.Second)
		got := d.lista()
		if len(got) != 2 ||
			got[0].Sub(inicio) != 2*time.Second ||
			got[1].Sub(inicio) != 5*time.Second {
			t.Fatalf("disparos %v depois do início", got)
		}
		cancel()
		if err := <-fim; !errors.Is(err, context.Canceled) {
			t.Fatalf("Run devolveu %v", err)
		}
	})
}

// livro:fim pump-synctest

// livro:inicio pump-virtual

// Com o relógio virtual, quem anda com o tempo é o teste. O pump roda
// numa goroutine de verdade, então o teste precisa de dois sinais
// explícitos: BlockUntil, para saber que o pump já pediu para ser
// acordado, e o canal feito, para saber que o disparo aconteceu.
func TestPumpComRelogioVirtual(t *testing.T) {
	v := clock.NewVirtual(time.Date(2026, 9, 25, 9, 0, 0, 0, time.UTC))
	feito := make(chan time.Time, 1)
	p := timer.NewPump(v, func(at time.Time) (int, error) {
		feito <- at
		return 1, nil
	})
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	go func() { _ = p.Run(ctx) }()

	prazo := v.Now().Add(time.Minute)
	if err := p.Schedule(ctx, prazo); err != nil {
		t.Fatal(err)
	}
	v.BlockUntil(1) // o pump criou o timer do prazo
	v.Advance(time.Minute)
	if at := <-feito; !at.Equal(prazo) {
		t.Fatalf("disparou em %v, prazo %v", at, prazo)
	}
}

// livro:fim pump-virtual
