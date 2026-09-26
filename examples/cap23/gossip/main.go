// Command gossip mede em quantas rodadas uma novidade chega a todos os
// nós quando cada nó conversa com um nó sorteado por rodada — a
// disseminação por piggyback do SWIM (Capítulo 23).
//
//	go run ./examples/cap23/gossip
//	go run ./examples/cap23/gossip -curva 1024 > curva.csv
//
// Simplificação: a novidade viaja em toda mensagem, sem o limite de
// retransmissões do SWIM, e nenhuma mensagem se perde.
package main

import (
	"flag"
	"fmt"
	"math"
	"math/rand/v2"
	"slices"
)

// livro:inicio gossip

// rodada faz cada nó mandar um ping a um nó sorteado, com a novidade de
// carona. Em push, só a mensagem de ida a leva; em push-pull, também a
// resposta: quem pergunta a um nó que já sabe aprende na volta.
func rodada(r *rand.Rand, sabe []bool, pull bool) {
	n := len(sabe)
	antes := slices.Clone(sabe) // todos agem sobre o estado da rodada
	for i := range n {
		j := r.IntN(n - 1)
		if j >= i {
			j++ // nunca a si mesmo
		}
		if antes[i] {
			sabe[j] = true
		}
		if pull && antes[j] {
			sabe[i] = true
		}
	}
}

// rodadas conta as rodadas até todos saberem, a partir de um nó só; e,
// se curva não for nil, anota a fração que sabe a cada rodada.
func rodadas(r *rand.Rand, n int, pull bool, curva *[]float64) int {
	sabe := make([]bool, n)
	sabe[0] = true
	for k := 1; ; k++ {
		rodada(r, sabe, pull)
		c := 0
		for _, s := range sabe {
			if s {
				c++
			}
		}
		if curva != nil {
			*curva = append(*curva, float64(c)/float64(n))
		}
		if c == n {
			return k
		}
	}
}

// livro:fim gossip

func main() {
	curva := flag.Int("curva", 0,
		"imprime a fração informada por rodada para este tamanho, "+
			"em CSV")
	flag.Parse()
	if *curva > 0 {
		imprimirCurva(*curva)
		return
	}
	const sementes = 200
	fmt.Printf("%6s %8s %14s %14s %16s\n", "nós", "log2(N)",
		"push (média)", "push (pior)", "push-pull (média)")
	for _, n := range []int{8, 16, 64, 256, 1024, 4096} {
		var somaP, somaPP, piorP int
		for s := range uint64(sementes) {
			r := rand.New(rand.NewPCG(s, uint64(n)))
			k := rodadas(r, n, false, nil)
			somaP += k
			piorP = max(piorP, k)
			somaPP += rodadas(r, n, true, nil)
		}
		fmt.Printf("%6d %8.0f %14.1f %14d %16.1f\n", n,
			math.Log2(float64(n)), float64(somaP)/sementes, piorP,
			float64(somaPP)/sementes)
	}
}

// imprimirCurva imprime a média, sobre 200 sementes, da fração de nós
// que sabem, rodada a rodada, em push e em push-pull. Uma semente que
// já terminou conta como 1 nas rodadas seguintes.
func imprimirCurva(n int) {
	const sementes, maximo = 200, 40
	var push, pp [maximo]float64
	for s := range uint64(sementes) {
		for _, pull := range []bool{false, true} {
			var c []float64
			rodadas(rand.New(rand.NewPCG(s, 7)), n, pull, &c)
			for k := range maximo {
				f := 1.0
				if k < len(c) {
					f = c[k]
				}
				if pull {
					pp[k] += f / sementes
				} else {
					push[k] += f / sementes
				}
			}
		}
	}
	fmt.Println("rodada,push,push_pull")
	for k := range maximo {
		fmt.Printf("%d,%.4f,%.4f\n", k+1, push[k], pp[k])
	}
}
