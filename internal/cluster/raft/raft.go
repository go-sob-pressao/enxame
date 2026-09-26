package raft

import (
	"errors"
	"math/rand/v2"
	"slices"
)

// NodeID identifica um nó do grupo Raft. Zero significa "nenhum".
type NodeID uint64

// Term é o mandato: um número que só cresce e ordena as lideranças.
type Term uint64

// Index é a posição de uma entrada no log, a partir de 1.
type Index uint64

// State é o papel do nó no termo corrente.
type State uint8

// Papéis de um nó.
const (
	Follower State = iota
	Candidate
	Leader
)

func (s State) String() string {
	return [...]string{"seguidor", "candidato", "líder"}[s]
}

// ErrNotLeader é devolvido a quem propõe a um nó que não é líder.
var ErrNotLeader = errors.New("raft: este nó não é o líder")

// Config configura um nó. Os tempos são contados em ticks: o nó não lê
// relógio nenhum, quem chama Tick decide quanto vale um tick.
type Config struct {
	ID    NodeID
	Peers []NodeID // todos os membros, inclusive este

	// ElectionTicks é a base do timeout de eleição; o valor efetivo é
	// sorteado em [ElectionTicks, 2*ElectionTicks) a cada reinício.
	ElectionTicks  int
	HeartbeatTicks int

	// Rand sorteia um inteiro em [0, n). Injetado para que a simulação
	// reproduza a mesma sequência a partir da mesma seed.
	Rand func(n int) int
}

// Node é um membro do grupo. Não é seguro para uso concorrente: quem o
// usa chama Tick, Step, Propose e Ready de uma goroutine só.
type Node struct {
	cfg Config

	state    State
	term     Term
	votedFor NodeID
	leader   NodeID

	electionElapsed  int
	electionTimeout  int
	heartbeatElapsed int
	votes            map[NodeID]bool

	// log[0] é uma sentinela: índice e termo do snapshot
	log      []Entry
	commit   Index
	entregue Index // até onde as entradas comitadas já saíram em Ready
	next     map[NodeID]Index
	match    map[NodeID]Index

	outbox []Message
}

// New cria um nó seguidor no termo zero.
func New(cfg Config) *Node {
	if cfg.Rand == nil {
		cfg.Rand = rand.IntN
	}
	cfg.Peers = slices.Clone(cfg.Peers)
	slices.Sort(cfg.Peers)
	n := &Node{cfg: cfg, log: []Entry{{}}}
	n.becomeFollower(0, 0)
	return n
}

// Status é um retrato do nó, para testes, métricas e o coordenador.
type Status struct {
	ID        NodeID
	State     State
	Term      Term
	Leader    NodeID
	Commit    Index
	LastIndex Index
}

// Status devolve o estado corrente.
func (n *Node) Status() Status {
	return Status{
		ID:        n.cfg.ID,
		State:     n.state,
		Term:      n.term,
		Leader:    n.leader,
		Commit:    n.commit,
		LastIndex: n.lastIndex(),
	}
}

// Ready é o que o nó produziu desde a última chamada: mensagens para
// entregar aos outros nós e entradas comitadas para a aplicação
// aplicar, em ordem.
type Ready struct {
	Messages  []Message
	Committed []Entry
}

// Ready entrega (e esquece) o que o nó produziu.
func (n *Node) Ready() Ready {
	r := Ready{Messages: n.outbox}
	n.outbox = nil
	if n.commit > n.entregue {
		r.Committed = n.slice(n.entregue+1, n.commit+1)
		n.entregue = n.commit
	}
	return r
}

func (n *Node) quorum() int { return len(n.cfg.Peers)/2 + 1 }

func (n *Node) send(m Message) {
	m.From, m.Term = n.cfg.ID, n.term
	n.outbox = append(n.outbox, m)
}

func (n *Node) others() []NodeID {
	outros := make([]NodeID, 0, len(n.cfg.Peers)-1)
	for _, p := range n.cfg.Peers {
		if p != n.cfg.ID {
			outros = append(outros, p)
		}
	}
	return outros
}
