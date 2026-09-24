package vazamentos

import (
	"context"
	"testing"

	"go.uber.org/goleak"
)

// Os testes abaixo PROVAM os vazamentos: as goroutines dessas quatro
// funções ficam vivas de propósito, e o TestMain as declara como
// esperadas.
func TestMain(m *testing.M) {
	goleak.VerifyTestMain(
		m,
		goleak.IgnoreAnyFunction(
			"github.com/go-sob-pressao/enxame/examples/cap07/vazamentos.PrimeiroResultado.func1",
		),
		goleak.IgnoreAnyFunction(
			"github.com/go-sob-pressao/enxame/examples/cap07/vazamentos.Consumir.func1",
		),
		goleak.IgnoreAnyFunction(
			"github.com/go-sob-pressao/enxame/examples/cap07/vazamentos.Vigiar.func1",
		),
		goleak.IgnoreAnyFunction(
			"github.com/go-sob-pressao/enxame/examples/cap07/vazamentos.Publicar.func1",
		),
	)
}

// vaza confirma que f deixa goroutines vivas: goleak.Find devolve erro.
func vaza(t *testing.T, f func()) {
	t.Helper()
	antes := goleak.IgnoreCurrent()
	f()
	if err := goleak.Find(antes); err == nil {
		t.Fatal("esperava vazamento, e não houve")
	} else {
		t.Logf("goleak: %.160s…", err)
	}
}

func TestEnvioSemReceptor(t *testing.T) {
	vaza(t, func() {
		PrimeiroResultado(
			[]func() int{
				func() int { return 1 },
				func() int { return 2 },
				func() int { return 3 },
			},
		)
	})
}

func TestRecepcaoSemEmissor(t *testing.T) {
	vaza(t, func() { Consumir(make(chan int), func(int) {}) })
}

func TestContextoIgnorado(t *testing.T) {
	vaza(t, func() {
		ctx, cancel := context.WithCancel(context.Background())
		Vigiar(ctx, func() {})
		cancel()
	})
}

func TestTickerNaoParado(t *testing.T) {
	vaza(t, func() { Publicar(func() {}) })
}

func TestCorrigidosNaoVazam(t *testing.T) {
	antes := goleak.IgnoreCurrent()
	PrimeiroResultadoCorrigido(
		[]func() int{func() int { return 1 }, func() int { return 2 }},
	)
	ctx, cancel := context.WithCancel(context.Background())
	VigiarCorrigido(ctx, func() {})
	cancel()
	goleak.VerifyNone(t, antes)
}
