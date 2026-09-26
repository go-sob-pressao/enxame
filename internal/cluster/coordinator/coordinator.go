package coordinator

import "context"

// NodeID identifica um nó do cluster.
type NodeID string

// NumPartitions é fixo na criação do cluster (ADR-010).
const NumPartitions = 512

// livro:inicio contrato

// Assignment é o mapa partição→nó numa época. Toda mudança do mapa cria
// uma época nova; quem recebe um mapa de época menor que o que já tem o
// descarta.
type Assignment struct {
	Epoch uint64
	// len == NumPartitions; Owners[p] é o dono da partição p
	Owners []NodeID
}

// Membership responde quem está no cluster, na visão deste nó: ele
// mesmo e os que ele não dá por mortos.
type Membership interface {
	Members(ctx context.Context) ([]NodeID, error)
}

// Coordinator responde, a qualquer momento: quem é o líder, quem está
// no cluster e de quem é cada partição. Declarado aqui, no consumidor;
// as implementações (pgcoord, raftcoord) não são conhecidas por este
// pacote.
type Coordinator interface {
	Membership
	Leader(ctx context.Context) (NodeID, error)
	Assignment(ctx context.Context) (Assignment, error)
	Close() error
}

// livro:fim contrato
