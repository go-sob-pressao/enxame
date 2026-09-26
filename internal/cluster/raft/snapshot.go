package raft

import (
	"errors"
	"fmt"
)

// Snapshot é o estado da aplicação até Index, inclusive: o que ela sabe
// reconstruir sozinha, sem as entradas do log que o produziram. O Raft
// não interpreta Data.
type Snapshot struct {
	Index Index
	Term  Term
	Data  []byte
}

// ErrCompactar é devolvido por Compact fora do intervalo permitido.
var ErrCompactar = errors.New("raft: índice de compactação inválido")

// livro:inicio compact

// Compact descarta o log até index, inclusive, trocando-o por um
// snapshot com o estado que a aplicação produziu até ali. Só entradas
// já entregues à aplicação podem virar snapshot.
func (n *Node) Compact(index Index, data []byte) error {
	if index <= n.base() || index > n.entregue {
		return fmt.Errorf("%w: %d fora de (%d, %d]",
			ErrCompactar, index, n.base(), n.entregue)
	}
	termo, _ := n.termAt(index)
	n.snapshot = &Snapshot{Index: index, Term: termo, Data: data}
	resto := n.slice(index+1, n.lastIndex()+1)
	n.log = append([]Entry{{Index: index, Term: termo}}, resto...)
	return nil
}

// livro:fim compact

// livro:inicio snapshot-envio

// sendSnapshot envia o snapshot ao seguidor que ficou para trás do
// começo do log: não há mais entradas para mandar a ele, só o estado.
func (n *Node) sendSnapshot(p NodeID) {
	n.send(Message{Type: MsgSnap, To: p, Snapshot: n.snapshot})
}

// handleSnap instala o snapshot do líder. Se o log local já tem a
// entrada em que o snapshot termina, com o mesmo termo, o que vem
// depois dela continua valendo; senão, o log inteiro é substituído.
func (n *Node) handleSnap(m Message) {
	n.becomeFollower(m.Term, m.From)
	s := m.Snapshot
	if s.Index <= n.commit {
		n.send(Message{Type: MsgAppResp, To: m.From, Success: true,
			Match: n.commit})
		return
	}
	var resto []Entry
	if t, ok := n.termAt(s.Index); ok && t == s.Term {
		resto = n.slice(s.Index+1, n.lastIndex()+1)
	}
	n.log = append([]Entry{{Index: s.Index, Term: s.Term}}, resto...)
	n.snapshot, n.instalar = s, s
	n.commit, n.entregue = s.Index, s.Index
	n.send(Message{Type: MsgAppResp, To: m.From, Success: true,
		Match: s.Index})
}

// livro:fim snapshot-envio
