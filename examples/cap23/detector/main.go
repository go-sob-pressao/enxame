// Command detector simula um dia de batidas de um nó com pausas de
// coleta de lixo e compara dois detectores: timeout fixo e phi accrual
// (Capítulo 23).
//
//	go run ./examples/cap23/detector
//	go run ./examples/cap23/detector -csv phi.csv
//
// O nó bate a cada 500 ms, com um pouco de variação. Em média a cada
// três minutos, ele para numa pausa de duração log-normal, com mediana
// de 1,5 s. Nenhuma pausa é uma morte: toda declaração de morte é um
// falso positivo. Depois, o nó morre de verdade em mil instantes
// sorteados, e se mede quanto cada detector demora a perceber.
package main

import (
	"flag"
	"fmt"
	"math"
	"math/rand/v2"
	"os"
	"time"

	"github.com/go-sob-pressao/enxame/internal/cluster/membership"
)

var t0 = time.Date(2026, 9, 26, 0, 0, 0, 0, time.UTC)

// livro:inicio simulacao-gc

// batidas gera os instantes das batidas de um dia: a cada intervalo,
// com o desvio dado e, se pausas, uma pausa a cada ~3 min, de duração
// log-normal com mediana de 1,5 s, que atrasa a batida seguinte.
func batidas(r *rand.Rand, intervalo, desvio time.Duration,
	pausas bool) []time.Time {
	var bs []time.Time
	for em := t0; em.Sub(t0) < 24*time.Hour; {
		bs = append(bs, em)
		passo := max(intervalo+time.Duration(r.NormFloat64()*
			float64(desvio)), 10*time.Millisecond)
		if pausas && r.Float64() < intervalo.Seconds()/180 {
			pausa := 1.5 * math.Exp(0.4*r.NormFloat64()) // segundos
			passo += time.Duration(pausa * float64(time.Second))
		}
		em = em.Add(passo)
	}
	return bs
}

// mortes conta quantas vezes o detector passa a dizer "morto" de um nó
// que só pausou, consultado a cada 100 ms.
func mortes(d membership.Detector, bs []time.Time) (n int,
	suspeita time.Duration) {
	antes := membership.Vivo
	for i, b := range bs {
		d.Batida(b)
		if i+1 == len(bs) {
			break
		}
		for t := b.Add(100 * time.Millisecond); t.Before(bs[i+1]); t =
			t.Add(100 * time.Millisecond) {
			e := d.Estado(t)
			if e == membership.Morto && antes != membership.Morto {
				n++
			}
			if e == membership.Suspeito {
				suspeita += 100 * time.Millisecond
			}
			antes = e
		}
		antes = membership.Vivo // a batida chegou: vivo de novo
	}
	return n, suspeita
}

// livro:fim simulacao-gc

// deteccao mede, para mortes em instantes sorteados, quanto tempo o
// detector demora a dizer "morto".
func deteccao(novo func() membership.Detector, bs []time.Time,
	r *rand.Rand) (media, pior time.Duration) {
	const mortes = 1000
	var soma time.Duration
	for range mortes {
		k := 1500 + r.IntN(len(bs)-1500) // morre depois da batida k
		d := novo()
		for _, b := range bs[max(0, k-1500) : k+1] {
			d.Batida(b)
		}
		t := bs[k]
		for d.Estado(t) != membership.Morto {
			t = t.Add(100 * time.Millisecond)
		}
		soma += t.Sub(bs[k])
		pior = max(pior, t.Sub(bs[k]))
	}
	return soma / mortes, pior
}

type detector struct {
	nome string
	novo func() membership.Detector
}

var detectores = []detector{
	{"timeout fixo de 2 s", fixo(2 * time.Second)},
	{"timeout fixo de 5 s", fixo(5 * time.Second)},
	{"phi, janela 100, folga 1 s", phi(100, time.Second)},
	{"phi, janela 1000, folga 1 s", phi(1000, time.Second)},
	{"phi, janela 1000, folga 3 s", phi(1000, 3*time.Second)},
}

func main() {
	csv := flag.String("csv", "", "grava intervalos e a curva de phi")
	semente := flag.Uint64("semente", 23, "semente do sorteio")
	flag.Parse()
	r := rand.New(rand.NewPCG(*semente, 0))

	fmt.Println("cenário 1: um nó, batidas a cada 500 ms, pausas de GC")
	bs := batidas(r, 500*time.Millisecond, 20*time.Millisecond, true)
	longos := 0
	for i := 1; i < len(bs); i++ {
		if bs[i].Sub(bs[i-1]) > 2*time.Second {
			longos++
		}
	}
	fmt.Printf("um dia: %d batidas; %d intervalos acima de 2 s\n",
		len(bs), longos)
	fmt.Printf("%-28s %14s %12s %16s %10s\n", "detector",
		"mortes falsas", "em suspeita", "detecção média", "pior")
	for _, d := range detectores {
		n, s := mortes(d.novo(), bs)
		media, pior := deteccao(d.novo, bs,
			rand.New(rand.NewPCG(*semente, 1)))
		fmt.Printf("%-28s %14d %12v %16v %10v\n", d.nome, n,
			s.Round(time.Second), media.Round(10*time.Millisecond),
			pior.Round(10*time.Millisecond))
	}

	fmt.Println()
	fmt.Println("cenário 2: dois nós sem pausas — um rápido " +
		"(500 ± 20 ms) e um lento (1,5 s ± 400 ms)")
	rapido := batidas(r, 500*time.Millisecond, 20*time.Millisecond,
		false)
	lento := batidas(r, 1500*time.Millisecond, 400*time.Millisecond,
		false)
	fmt.Printf("%-28s %14s %14s %16s %16s\n", "detector",
		"falsas rápido", "falsas lento", "detecção rápido",
		"detecção lento")
	for _, d := range detectores {
		nr, _ := mortes(d.novo(), rapido)
		nl, _ := mortes(d.novo(), lento)
		mr, _ := deteccao(d.novo, rapido, rand.New(rand.NewPCG(1, 2)))
		ml, _ := deteccao(d.novo, lento, rand.New(rand.NewPCG(1, 3)))
		fmt.Printf("%-28s %14d %14d %16v %16v\n", d.nome, nr, nl,
			mr.Round(10*time.Millisecond),
			ml.Round(10*time.Millisecond))
	}
	if *csv != "" {
		if err := gravar(*csv, bs); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}
}

func fixo(limite time.Duration) func() membership.Detector {
	return func() membership.Detector {
		return &membership.Fixo{Limite: limite}
	}
}

// gravar escreve os intervalos do dia e, para um detector que viu as
// últimas 100 batidas, phi em função do silêncio.
func gravar(caminho string, bs []time.Time) error {
	f, err := os.Create(caminho)
	if err != nil {
		return err
	}
	fmt.Fprintln(f, "tipo,x,y")
	for i := 1; i < len(bs); i++ {
		fmt.Fprintf(f, "intervalo,%.4f,\n",
			bs[i].Sub(bs[i-1]).Seconds())
	}
	for _, janela := range []struct {
		nome string
		bs   []time.Time
	}{{"phi-dia", bs}, {"phi-regular", regulares()}} {
		p := membership.NovoPhi(500 * time.Millisecond)
		for _, b := range janela.bs[len(janela.bs)-101:] {
			p.Batida(b)
		}
		ultima := janela.bs[len(janela.bs)-1]
		for ms := 0; ms <= 8000; ms += 50 {
			d := time.Duration(ms) * time.Millisecond
			fmt.Fprintf(f, "%s,%.3f,%.3f\n", janela.nome, d.Seconds(),
				min(p.Valor(ultima.Add(d)), 20))
		}
	}
	return f.Close()
}

// phi cria um detector phi accrual com a janela e a folga dadas.
func phi(janela int, folga time.Duration) func() membership.Detector {
	return func() membership.Detector {
		p := membership.NovoPhi(500 * time.Millisecond)
		p.Janela, p.Folga = janela, folga
		return p
	}
}

// regulares são batidas a cada 500 ms, sem pausa nenhuma.
func regulares() []time.Time {
	var bs []time.Time
	for i := range 200 {
		bs = append(bs, t0.Add(time.Duration(i)*500*time.Millisecond))
	}
	return bs
}
