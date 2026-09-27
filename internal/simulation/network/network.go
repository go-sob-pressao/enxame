package network

import (
	"time"

	"github.com/go-sob-pressao/enxame/internal/simulation/scheduler"
)

// Config descreve o quanto a rede mente.
type Config struct {
	AtrasoMin, AtrasoMax time.Duration // atraso sorteado por mensagem
	Perda                float64       // probabilidade de sumir
	// probabilidade de chegar duas vezes
	Duplicacao float64
}

// Stats conta o que aconteceu com as mensagens.
type Stats struct {
	Enviadas, Entregues, Perdidas, Duplicadas, Cortadas int
}

// livro:inicio network

// Network entrega mensagens entre nós identificados por inteiros, com
// atraso sorteado (e, portanto, fora de ordem), perda, duplicação e
// partição. Todo sorteio vem do escalonador: a rede é tão
// determinística quanto a seed.
type Network[M any] struct {
	s       *scheduler.Scheduler
	cfg     Config
	entrega func(de, para int, m M)
	grupo   map[int]int
	Stats   Stats
}

// New cria a rede; entrega é chamada quando uma mensagem chega.
func New[M any](s *scheduler.Scheduler, cfg Config,
	entrega func(de, para int, m M)) *Network[M] {
	return &Network[M]{
		s:       s,
		cfg:     cfg,
		entrega: entrega,
		grupo:   map[int]int{},
	}
}

// Send despacha m de de para para. A partição é conferida no envio e de
// novo na chegada: uma mensagem em voo não atravessa uma partição que
// se formou no caminho.
func (n *Network[M]) Send(de, para int, m M) {
	n.Stats.Enviadas++
	r := n.s.Rand()
	if r.Float64() < n.cfg.Perda {
		n.Stats.Perdidas++
		return
	}
	copias := 1
	if r.Float64() < n.cfg.Duplicacao {
		copias = 2
		n.Stats.Duplicadas++
	}
	for range copias {
		faixa := int64(n.cfg.AtrasoMax - n.cfg.AtrasoMin)
		atraso := n.cfg.AtrasoMin + time.Duration(r.Int64N(faixa+1))
		n.s.After(atraso, func() {
			if n.grupo[de] != n.grupo[para] {
				n.Stats.Cortadas++
				return
			}
			n.Stats.Entregues++
			n.entrega(de, para, m)
		})
	}
}

// livro:fim network

// Partition define a topologia: nós no mesmo grupo se falam. Nós que
// não aparecem em nenhum grupo ficam juntos, no grupo zero.
func (n *Network[M]) Partition(grupos ...[]int) {
	clear(n.grupo)
	for g, ids := range grupos {
		for _, id := range ids {
			n.grupo[id] = g + 1
		}
	}
}

// Isolar põe o nó id num grupo só dele.
func (n *Network[M]) Isolar(id int) { n.grupo[id] = -id }

// Heal junta todos de novo.
func (n *Network[M]) Heal() { clear(n.grupo) }
