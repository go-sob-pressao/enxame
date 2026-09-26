package raftcoord

import (
	"sync"
	"time"

	"github.com/go-sob-pressao/enxame/internal/cluster/raft"
)

// Rede entrega mensagens Raft entre coordenadores do mesmo processo,
// com latência fixa. Serve aos testes e à bancada de medição; o
// transporte entre processos (gRPC) entra no Capítulo 18.
type Rede struct {
	Latencia time.Duration

	mu       sync.Mutex
	caixas   map[raft.NodeID]chan raft.Message
	parados  map[raft.NodeID]bool
	Enviadas int
}

// NovaRede cria uma rede vazia.
func NovaRede(latencia time.Duration) *Rede {
	return &Rede{
		Latencia: latencia,
		caixas:   map[raft.NodeID]chan raft.Message{},
		parados:  map[raft.NodeID]bool{},
	}
}

func (r *Rede) registrar(id raft.NodeID) <-chan raft.Message {
	r.mu.Lock()
	defer r.mu.Unlock()
	c := make(chan raft.Message, 4096)
	r.caixas[id] = c
	delete(r.parados, id)
	return c
}

// Parar tira o nó da rede: nada chega a ele nem sai dele.
func (r *Rede) Parar(id raft.NodeID) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.parados[id] = true
}

func (r *Rede) enviar(m raft.Message) {
	r.mu.Lock()
	r.Enviadas++
	r.mu.Unlock()
	time.AfterFunc(r.Latencia, func() {
		r.mu.Lock()
		defer r.mu.Unlock()
		c, ok := r.caixas[m.To]
		if !ok || r.parados[m.To] || r.parados[m.From] {
			return
		}
		select {
		case c <- m:
		// caixa cheia: a mensagem se perde, como na rede de verdade
		default:
		}
	})
}
