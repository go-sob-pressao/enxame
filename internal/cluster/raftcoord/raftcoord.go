package raftcoord

import (
	"context"
	"encoding/json"
	"slices"
	"sync"
	"time"

	"github.com/go-sob-pressao/enxame/internal/cluster/coordinator"
	"github.com/go-sob-pressao/enxame/internal/cluster/raft"
)

// Config configura um coordenador.
type Config struct {
	ID             coordinator.NodeID
	Peers          []coordinator.NodeID // todos, inclusive este
	Tick           time.Duration        // quanto vale um tick do Raft
	ElectionTicks  int
	HeartbeatTicks int
	Rede           *Rede
}

// livro:inicio raftcoord

// Coord implementa coordinator.Coordinator sobre o Raft do Capítulo 25.
// O estado replicado é só o mapa partição→nó: cada mudança é uma
// entrada do log, proposta pelo líder quando o conjunto de membros
// ativos muda.
type Coord struct {
	cfg     Config
	porID   map[coordinator.NodeID]raft.NodeID
	porRaft map[raft.NodeID]coordinator.NodeID

	mu    sync.Mutex
	node  *raft.Node
	mapa  coordinator.Assignment
	lider coordinator.NodeID

	parar context.CancelFunc
	fim   chan struct{}
}

// New inicia o coordenador; ele roda até Close.
func New(cfg Config) *Coord {
	peers := slices.Sorted(slices.Values(cfg.Peers))
	c := &Coord{
		cfg:     cfg,
		porID:   map[coordinator.NodeID]raft.NodeID{},
		porRaft: map[raft.NodeID]coordinator.NodeID{},
		fim:     make(chan struct{}),
	}
	var ids []raft.NodeID
	for i, p := range peers {
		c.porID[p], c.porRaft[raft.NodeID(i+1)] = raft.NodeID(i+1), p
		ids = append(ids, raft.NodeID(i+1))
	}
	c.node = raft.New(raft.Config{
		ID: c.porID[cfg.ID], Peers: ids,
		ElectionTicks:  cfg.ElectionTicks,
		HeartbeatTicks: cfg.HeartbeatTicks,
	})
	caixa := cfg.Rede.registrar(c.porID[cfg.ID])
	ctx, parar := context.WithCancel(context.Background())
	c.parar = parar
	go c.laco(ctx, caixa)
	return c
}

// laco é a única goroutine que toca o nó Raft: ticks, mensagens e o que
// o nó produz passam por aqui, um de cada vez.
func (c *Coord) laco(ctx context.Context, caixa <-chan raft.Message) {
	defer close(c.fim)
	t := time.NewTicker(c.cfg.Tick)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			c.mu.Lock()
			c.node.Tick()
			c.talvezRedistribuir()
			c.processar()
			c.mu.Unlock()
		case m := <-caixa:
			c.mu.Lock()
			c.node.Step(m)
			c.processar()
			c.mu.Unlock()
		}
	}
}

// talvezRedistribuir: no líder, se os membros ativos não são os donos
// do mapa corrente, propõe um mapa novo.
func (c *Coord) talvezRedistribuir() {
	st := c.node.Status()
	if st.State != raft.Leader {
		return
	}
	ativos := make([]coordinator.NodeID, 0, len(st.Ativos))
	for _, id := range st.Ativos {
		ativos = append(ativos, c.porRaft[id])
	}
	if slices.Equal(
		donos(c.mapa),
		slices.Sorted(slices.Values(ativos)),
	) {
		return
	}
	novo := coordinator.Distribute(ativos, c.mapa)
	dados, _ := json.Marshal(novo)
	_, _ = c.node.Propose(dados)
}

// processar envia as mensagens e aplica as entradas comitadas.
func (c *Coord) processar() {
	r := c.node.Ready()
	for _, m := range r.Messages {
		c.cfg.Rede.enviar(m)
	}
	if r.Snapshot != nil {
		c.aplicar(r.Snapshot.Data)
	}
	for _, e := range r.Committed {
		if e.Data != nil {
			c.aplicar(e.Data)
		}
	}
	if len(r.Committed) > 0 {
		c.compactar()
	}
	st := c.node.Status()
	c.lider = c.porRaft[st.Leader]
}

// livro:fim raftcoord

func (c *Coord) aplicar(dados []byte) {
	var a coordinator.Assignment
	if json.Unmarshal(dados, &a) == nil && a.Epoch > c.mapa.Epoch {
		c.mapa = a
	}
}

// compactar troca o log inteiro por um snapshot: o estado é só o mapa.
func (c *Coord) compactar() {
	st := c.node.Status()
	if st.Commit < 64 {
		return
	}
	dados, _ := json.Marshal(c.mapa)
	_ = c.node.Compact(st.Commit, dados)
}

func donos(a coordinator.Assignment) []coordinator.NodeID {
	return slices.Compact(slices.Sorted(slices.Values(a.Owners)))
}

// Leader devolve o líder que este nó conhece.
func (c *Coord) Leader(context.Context) (coordinator.NodeID, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.lider, nil
}

// Members devolve os donos do mapa corrente.
func (c *Coord) Members(context.Context) ([]coordinator.NodeID, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return donos(c.mapa), nil
}

// Assignment devolve o mapa corrente deste nó.
func (c *Coord) Assignment(
	context.Context,
) (coordinator.Assignment, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	a := c.mapa
	a.Owners = slices.Clone(a.Owners)
	return a, nil
}

// Close para o coordenador e espera a goroutine dele terminar.
func (c *Coord) Close() error {
	c.parar()
	<-c.fim
	c.cfg.Rede.Parar(c.porID[c.cfg.ID])
	return nil
}
