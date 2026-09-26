// Package rafttest — um grupo Raft em memória, determinístico, para
// testes: entrega de mensagens em ordem fixa, queda, partição e
// registro das violações de segurança observadas.
package rafttest

import (
	"errors"
	"fmt"
	"math/rand/v2"
	"slices"

	"github.com/go-sob-pressao/enxame/internal/cluster/raft"
)

// Cluster é um grupo de nós Raft ligados por uma rede de mentira.
type Cluster struct {
	nodes map[raft.NodeID]*raft.Node
	ids   []raft.NodeID
	fila  []raft.Message
	fora  map[raft.NodeID]bool
	corte map[[2]raft.NodeID]bool
	bloq  map[bloqueio]bool

	lideres    map[raft.Term]raft.NodeID
	aplicados  map[raft.NodeID][]raft.Entry
	porIndice  map[raft.Index]raft.Entry
	violacoes  []string
	Entregues  int // mensagens entregues
	Descartes  int // mensagens descartadas por queda ou partição
	TicksDados int
}

// Opcoes ajusta os tempos do grupo.
type Opcoes struct {
	ElectionTicks  int
	HeartbeatTicks int
}

// New cria n nós (IDs 1..n) cujo sorteio de timeout vem de uma única
// seed.
func New(n int, seed uint64, o Opcoes) *Cluster {
	if o.ElectionTicks == 0 {
		o.ElectionTicks = 10
	}
	if o.HeartbeatTicks == 0 {
		o.HeartbeatTicks = 2
	}
	rng := rand.New(rand.NewPCG(seed, seed^0x9e3779b97f4a7c15))
	c := &Cluster{
		nodes:     map[raft.NodeID]*raft.Node{},
		fora:      map[raft.NodeID]bool{},
		corte:     map[[2]raft.NodeID]bool{},
		bloq:      map[bloqueio]bool{},
		lideres:   map[raft.Term]raft.NodeID{},
		aplicados: map[raft.NodeID][]raft.Entry{},
		porIndice: map[raft.Index]raft.Entry{},
	}
	for i := 1; i <= n; i++ {
		c.ids = append(c.ids, raft.NodeID(i))
	}
	for _, id := range c.ids {
		c.nodes[id] = raft.New(raft.Config{
			ID: id, Peers: c.ids,
			ElectionTicks:  o.ElectionTicks,
			HeartbeatTicks: o.HeartbeatTicks,
			Rand:           rng.IntN,
		})
	}
	return c
}

// Node devolve o nó id.
func (c *Cluster) Node(id raft.NodeID) *raft.Node { return c.nodes[id] }

// IDs devolve os identificadores em ordem.
func (c *Cluster) IDs() []raft.NodeID { return slices.Clone(c.ids) }

// Tick avança um tick em cada nó ativo e entrega tudo o que for gerado.
func (c *Cluster) Tick() {
	c.TicksDados++
	for _, id := range c.ids {
		if !c.fora[id] {
			c.nodes[id].Tick()
			c.coletar(id)
		}
	}
	c.entregar()
}

// Run executa n ticks.
func (c *Cluster) Run(n int) {
	for range n {
		c.Tick()
	}
}

func (c *Cluster) coletar(id raft.NodeID) {
	r := c.nodes[id].Ready()
	c.fila = append(c.fila, r.Messages...)
	c.aplicar(id, r.Committed)
	c.registrarLideres()
}

// aplicar registra as entradas comitadas de um nó e verifica a
// segurança da máquina de estados: dois nós nunca aplicam entradas
// diferentes no mesmo índice.
func (c *Cluster) aplicar(id raft.NodeID, es []raft.Entry) {
	for _, e := range es {
		outra, ok := c.porIndice[e.Index]
		difere := ok && (outra.Term != e.Term ||
			string(outra.Data) != string(e.Data))
		if difere {
			c.violacoes = append(c.violacoes, fmt.Sprintf(
				"índice %d aplicado com %q (termo %d) e %q (termo %d)",
				e.Index, outra.Data, outra.Term, e.Data, e.Term))
			continue
		}
		c.porIndice[e.Index] = e
		c.aplicados[id] = append(c.aplicados[id], e)
	}
}

// Aplicados devolve os comandos que o nó aplicou, em ordem, sem as
// entradas vazias de liderança.
func (c *Cluster) Aplicados(id raft.NodeID) [][]byte {
	var out [][]byte
	for _, e := range c.aplicados[id] {
		if e.Data != nil {
			out = append(out, e.Data)
		}
	}
	return out
}

// Campaign força o nó a se candidatar agora e entrega o que isso gerar.
func (c *Cluster) Campaign(id raft.NodeID) {
	c.nodes[id].Campaign()
	c.coletar(id)
	c.entregar()
}

// ProposeTo entrega o comando ao nó id, seja ele líder ou não.
func (c *Cluster) ProposeTo(id raft.NodeID, data []byte) {
	_, _ = c.nodes[id].Propose(data)
	c.coletar(id)
	c.entregar()
}

// ErrSemLider é devolvido por Propose quando não há líder.
var ErrSemLider = errors.New("rafttest: nenhum líder")

// Propose entrega o comando ao líder corrente.
func (c *Cluster) Propose(data []byte) error {
	l := c.Leader()
	if l == 0 {
		return ErrSemLider
	}
	_, err := c.nodes[l].Propose(data)
	c.coletar(l)
	c.entregar()
	return err
}

func (c *Cluster) entregar() {
	for len(c.fila) > 0 {
		m := c.fila[0]
		c.fila = c.fila[1:]
		if c.fora[m.From] || c.fora[m.To] ||
			c.corte[[2]raft.NodeID{m.From, m.To}] ||
			c.bloq[bloqueio{m.From, m.Type}] {
			c.Descartes++
			continue
		}
		c.Entregues++
		c.nodes[m.To].Step(m)
		c.coletar(m.To)
	}
}

// registrarLideres verifica a segurança de eleição: no máximo um líder
// por termo, ao longo de toda a execução.
func (c *Cluster) registrarLideres() {
	for _, id := range c.ids {
		s := c.nodes[id].Status()
		if s.State != raft.Leader {
			continue
		}
		if outro, ok := c.lideres[s.Term]; ok && outro != id {
			c.violacoes = append(
				c.violacoes,
				fmt.Sprintf(
					"dois líderes no termo %d: %d e %d",
					s.Term,
					outro,
					id,
				),
			)
			continue
		}
		c.lideres[s.Term] = id
	}
}

// Leader devolve o líder do maior termo entre os nós ativos, ou 0.
func (c *Cluster) Leader() raft.NodeID {
	var lider raft.NodeID
	var termo raft.Term
	for _, id := range c.ids {
		s := c.nodes[id].Status()
		if !c.fora[id] && s.State == raft.Leader && s.Term >= termo {
			lider, termo = id, s.Term
		}
	}
	return lider
}

// Crash para o nó: não recebe ticks, e mensagens de e para ele se
// perdem.
func (c *Cluster) Crash(id raft.NodeID) { c.fora[id] = true }

// Recover devolve o nó ao grupo, com o estado que tinha.
func (c *Cluster) Recover(id raft.NodeID) { delete(c.fora, id) }

// Partition define a topologia inteira: nós no mesmo grupo se falam,
// nós em grupos diferentes não. Substitui a partição anterior; um nó
// que não aparece em nenhum grupo fica isolado.
func (c *Cluster) Partition(grupos ...[]raft.NodeID) {
	clear(c.corte)
	grupo := map[raft.NodeID]int{}
	for g, ids := range grupos {
		for _, id := range ids {
			grupo[id] = g + 1
		}
	}
	for _, a := range c.ids {
		for _, b := range c.ids {
			if a != b && (grupo[a] == 0 || grupo[a] != grupo[b]) {
				c.corte[[2]raft.NodeID{a, b}] = true
			}
		}
	}
}

// Heal desfaz todas as partições e bloqueios.
func (c *Cluster) Heal() {
	clear(c.corte)
	clear(c.bloq)
}

type bloqueio struct {
	de   raft.NodeID
	tipo raft.MessageType
}

// Bloquear descarta toda mensagem do tipo enviada pelo nó de. Serve aos
// roteiros exatos: um líder que recebe os votos, mas cujo AppendEntries
// nunca chega.
func (c *Cluster) Bloquear(de raft.NodeID, tipo raft.MessageType) {
	c.bloq[bloqueio{de, tipo}] = true
}

// Violations devolve as violações de segurança observadas.
func (c *Cluster) Violations() []string {
	return slices.Clone(c.violacoes)
}
