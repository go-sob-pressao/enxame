package raft

// livro:inicio propose

// Propose acrescenta um comando ao log do líder e devolve o índice que
// ele ocupa. Estar no log não é estar comitado.
func (n *Node) Propose(data []byte) (Index, error) {
	if n.state != Leader {
		return 0, ErrNotLeader
	}
	i := n.lastIndex() + 1
	n.log = append(n.log, Entry{Term: n.term, Index: i, Data: data})
	n.match[n.cfg.ID] = i
	n.maybeCommit()
	n.broadcastAppend()
	return i, nil
}

// livro:fim propose

func (n *Node) broadcastAppend() {
	for _, p := range n.others() {
		n.sendAppend(p)
	}
}

// livro:inicio send-append

// sendAppend envia a um seguidor tudo o que ele ainda não tem, junto
// com a entrada imediatamente anterior: é por ela que o seguidor
// confere se os dois logs coincidem até ali. Sem entradas novas, é um
// heartbeat.
func (n *Node) sendAppend(p NodeID) {
	anterior := n.next[p] - 1
	termoAnterior, _ := n.termAt(anterior)
	n.send(Message{
		Type:      MsgApp,
		To:        p,
		PrevIndex: anterior,
		PrevTerm:  termoAnterior,
		Entries:   n.slice(anterior+1, n.lastIndex()+1),
		Commit:    n.commit,
	})
}

// livro:fim send-append

// livro:inicio handle-app

// handleApp é o seguidor recebendo AppendEntries. Se a entrada anterior
// não coincide, recusa e dá uma dica; se coincide, descarta o que
// conflitar com o líder, acrescenta o que faltar e avança o commit.
func (n *Node) handleApp(m Message) {
	n.becomeFollower(m.Term, m.From)
	if t, ok := n.termAt(m.PrevIndex); !ok || t != m.PrevTerm {
		dica := min(n.lastIndex(), m.PrevIndex-1)
		n.send(Message{Type: MsgAppResp, To: m.From, Hint: dica})
		return
	}
	for _, e := range m.Entries {
		if t, ok := n.termAt(e.Index); ok && t == e.Term {
			continue // já tenho esta entrada
		}
		if e.Index <= n.lastIndex() {
			// conflito: descarta desta entrada em diante
			n.log = n.log[:e.Index-n.base()]
		}
		n.log = append(n.log, e)
	}
	ultimaNova := m.PrevIndex + Index(len(m.Entries))
	if m.Commit > n.commit {
		n.commit = min(m.Commit, ultimaNova)
	}
	n.send(Message{
		Type:    MsgAppResp,
		To:      m.From,
		Success: true,
		Match:   ultimaNova,
	})
}

// livro:fim handle-app

// livro:inicio handle-app-resp

// handleAppResp é o líder recebendo a resposta. Aceitou: avança match e
// next e tenta comitar. Recusou: recua next até a dica e tenta de novo.
func (n *Node) handleAppResp(m Message) {
	if n.state != Leader {
		return
	}
	if m.Success {
		n.match[m.From] = max(n.match[m.From], m.Match)
		n.next[m.From] = n.match[m.From] + 1
		n.maybeCommit()
		if n.next[m.From] <= n.lastIndex() {
			n.sendAppend(m.From)
		}
		return
	}
	n.next[m.From] = max(1, min(n.next[m.From]-1, m.Hint+1))
	n.sendAppend(m.From)
}

// livro:fim handle-app-resp
