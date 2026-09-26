package raft

// Entry é uma entrada do log: um comando da aplicação, com o termo em
// que o líder a recebeu.
type Entry struct {
	Term  Term
	Index Index
	Data  []byte
}

// O log guarda log[0] como sentinela: a última entrada absorvida pelo
// snapshot (índice e termo zero antes do primeiro snapshot). A entrada
// de índice i fica em log[i-base].

func (n *Node) base() Index { return n.log[0].Index }

func (n *Node) lastIndex() Index {
	return n.base() + Index(len(n.log)-1)
}
func (n *Node) lastTerm() Term { return n.log[len(n.log)-1].Term }

// termAt devolve o termo da entrada i; ok é falso se i está além do fim
// ou já foi absorvida pelo snapshot.
func (n *Node) termAt(i Index) (Term, bool) {
	if i < n.base() || i > n.lastIndex() {
		return 0, false
	}
	return n.log[i-n.base()].Term, true
}

// slice devolve uma cópia das entradas de [de, ate).
func (n *Node) slice(de, ate Index) []Entry {
	out := make([]Entry, ate-de)
	copy(out, n.log[de-n.base():ate-n.base()])
	return out
}

// Log devolve uma cópia das entradas do log, sem a sentinela. Para
// testes e para a verificação de invariantes da simulação.
func (n *Node) Log() []Entry {
	return n.slice(n.base()+1, n.lastIndex()+1)
}
