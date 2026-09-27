package storefault

import (
	"errors"
	"math/rand/v2"
)

// ErrSemResposta é o que o nó recebe quando a instrução, ou a resposta
// dela, se perdeu: ele não sabe se ela aconteceu.
var ErrSemResposta = errors.New("storefault: sem resposta do banco")

// Stats conta o que aconteceu com as instruções.
type Stats struct {
	Enviadas, PerdidasNaIda, PerdidasNaVolta, Cortadas int
}

// livro:inicio conexao

// Conexao é o caminho de um nó até o banco, com as três falhas que
// importam: a instrução perdida na ida (não aconteceu), a resposta
// perdida na volta (aconteceu, e o nó recebe um erro mesmo assim) e o
// corte, em que nada passa. Todo sorteio vem de Rand, a fonte da seed.
type Conexao struct {
	Rand    *rand.Rand
	Perda   float64 // probabilidade de perder a ida, e de novo a volta
	Cortada bool
	Stats   Stats
}

// Fazer executa op no banco, se a instrução chegar, e diz ao nó se a
// resposta voltou.
func (c *Conexao) Fazer(op func()) error {
	c.Stats.Enviadas++
	if c.Cortada {
		c.Stats.Cortadas++
		return ErrSemResposta
	}
	if c.Rand.Float64() < c.Perda {
		c.Stats.PerdidasNaIda++
		return ErrSemResposta
	}
	op()
	if c.Rand.Float64() < c.Perda {
		c.Stats.PerdidasNaVolta++
		return ErrSemResposta
	}
	return nil
}

// livro:fim conexao
