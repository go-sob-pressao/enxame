// Package pool — quando o sync.Pool ajuda e quando não (Capítulo 29).
package pool

import (
	"bytes"
	"runtime"
	"sync"
	"testing"
)

var buffers = sync.Pool{New: func() any { return new(bytes.Buffer) }}

var sumidouro int

// livro:inicio pool-curto

// Vida curta: o buffer é pego, usado e devolvido dentro da mesma
// chamada. É o caso para que o sync.Pool existe.
func BenchmarkBufferCurtoSemPool(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		var buf bytes.Buffer
		buf.Grow(4096)
		buf.WriteString("resposta")
		sumidouro += buf.Len()
	}
}

func BenchmarkBufferCurtoComPool(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		buf := buffers.Get().(*bytes.Buffer)
		buf.Reset()
		buf.Grow(4096)
		buf.WriteString("resposta")
		sumidouro += buf.Len()
		buffers.Put(buf)
	}
}

// livro:fim pool-curto

// modelo é um objeto caro de montar — um template compilado, um
// esquema validado — que alguém quis guardar para reusar.
type modelo struct{ compilado [4096]byte }

// livro:inicio pool-cache

// O pool como cache: guarda mil modelos e, depois de n coletas, conta
// quantos o Get ainda devolve sem chamar New. Uma coleta move o
// conteúdo do pool para a reserva (o victim cache); a segunda descarta
// a reserva.
func TestPoolNaoECache(t *testing.T) {
	for _, coletas := range []int{0, 1, 2} {
		criados := 0
		p := sync.Pool{New: func() any { criados++; return new(modelo) }}
		for range 1000 {
			p.Put(new(modelo))
		}
		for range coletas {
			runtime.GC()
		}
		for range 1000 {
			sumidouro += len(p.Get().(*modelo).compilado)
		}
		t.Logf("%d coleta(s): %d de 1000 modelos precisaram ser "+
			"criados de novo", coletas, criados)
		if coletas == 2 && criados < 900 {
			t.Errorf("o pool guardou modelos por duas coletas")
		}
	}
}

// livro:fim pool-cache

// sessao fica com quem a pegou por muito tempo, atravessando coletas,
// e volta ao pool no fim: aí o pool ajuda, pelo victim cache.
type sessao struct{ _ [4096]byte }

var sessoes = sync.Pool{New: func() any { return new(sessao) }}

func BenchmarkSessaoLongaComPool(b *testing.B) {
	b.ReportAllocs()
	vivas := make([]*sessao, 0, 1000)
	for b.Loop() {
		s := sessoes.Get().(*sessao)
		vivas = append(vivas, s)
		if len(vivas) == cap(vivas) { // a sessão acabou, muito depois
			runtime.GC()
			for _, v := range vivas {
				sessoes.Put(v)
			}
			vivas = vivas[:0]
		}
	}
}

func BenchmarkSessaoLongaSemPool(b *testing.B) {
	b.ReportAllocs()
	vivas := make([]*sessao, 0, 1000)
	for b.Loop() {
		vivas = append(vivas, new(sessao))
		if len(vivas) == cap(vivas) {
			runtime.GC()
			vivas = vivas[:0]
		}
	}
}
