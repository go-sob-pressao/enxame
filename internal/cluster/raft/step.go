package raft

// livro:inicio step

// Step processa uma mensagem recebida de outro nó. A primeira regra
// vale para qualquer mensagem: termo maior que o meu significa que o
// mundo andou sem mim, e eu volto a ser seguidor nesse termo. Termo
// menor significa remetente atrasado, e a mensagem é descartada.
func (n *Node) Step(m Message) {
	switch {
	case m.Term > n.term:
		lider := NodeID(0)
		if m.Type == MsgApp {
			lider = m.From
		}
		n.becomeFollower(m.Term, lider)
	case m.Term < n.term:
		return
	}

	switch m.Type {
	case MsgVote:
		n.handleVote(m)
	case MsgVoteResp:
		n.handleVoteResp(m)
	case MsgApp:
		n.handleApp(m)
	case MsgAppResp:
	}
}

// livro:fim step

// handleApp, na etapa de eleição, só reconhece o líder e reinicia o
// timer: é o heartbeat.
func (n *Node) handleApp(m Message) {
	n.becomeFollower(m.Term, m.From)
	n.send(Message{Type: MsgAppResp, To: m.From})
}
