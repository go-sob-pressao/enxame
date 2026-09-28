package coordinator

import (
	"context"
	"errors"
	"slices"

	"github.com/go-sob-pressao/enxame/internal/core/id"
)

// NodeID identifica um nó do cluster.
type NodeID string

// NumPartitions é fixo na criação do cluster (ADR-010).
const NumPartitions = id.NumParticoes

// livro:inicio contrato

// Assignment é o mapa partição→nó numa época. Toda mudança do mapa cria
// uma época nova; quem recebe um mapa de época menor que o que já tem o
// descarta.
type Assignment struct {
	Epoch uint64
	// len == NumPartitions; Owners[p] é o dono da partição p
	Owners []NodeID
}

// ErrSemVisao é a resposta de um Membership que ainda não olhou o
// cluster: "só eu" seria uma resposta falsa (Cap. 32).
var ErrSemVisao = errors.New("coordinator: membership ainda sem visão")

// Membership responde quem está no cluster, na visão deste nó: ele
// mesmo e os que ele não dá por mortos — ou ErrSemVisao, antes da
// primeira leitura.
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

// livro:inicio distribuir

// Distribute calcula o mapa para os membros dados, mexendo no mínimo de
// partições possível em relação ao mapa anterior: quem continua no
// cluster mantém o que tinha até a sua cota; só as partições de quem
// saiu e o excesso de quem passou da cota mudam de dono. É uma função
// pura.
func Distribute(membros []NodeID, anterior Assignment) Assignment {
	membros = slices.Sorted(slices.Values(membros))
	novo := Assignment{
		Epoch:  anterior.Epoch + 1,
		Owners: make([]NodeID, NumPartitions),
	}
	if len(membros) == 0 {
		return novo
	}
	cota := map[NodeID]int{}
	for i, m := range membros {
		cota[m] = NumPartitions / len(membros)
		if i < NumPartitions%len(membros) {
			cota[m]++
		}
	}
	var orfas []int
	for p := range NumPartitions {
		dono := NodeID("")
		if len(anterior.Owners) == NumPartitions {
			dono = anterior.Owners[p]
		}
		if cota[dono] > 0 {
			novo.Owners[p] = dono
			cota[dono]--
			continue
		}
		orfas = append(orfas, p)
	}
	for _, p := range orfas {
		for _, m := range membros {
			if cota[m] > 0 {
				novo.Owners[p] = m
				cota[m]--
				break
			}
		}
	}
	return novo
}

// livro:fim distribuir

// Moved conta quantas partições mudaram de dono entre dois mapas.
func Moved(a, b Assignment) int {
	n := 0
	for p := range min(len(a.Owners), len(b.Owners)) {
		if a.Owners[p] != b.Owners[p] {
			n++
		}
	}
	return n
}
