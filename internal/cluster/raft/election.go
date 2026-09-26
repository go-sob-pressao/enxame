package raft

// livro:inicio tick

// Tick avança o relógio lógico do nó. O líder conta até o próximo
// heartbeat; os demais contam até desistir de esperar e se candidatar.
func (n *Node) Tick() {
	if n.state == Leader {
		n.heartbeatElapsed++
		if n.heartbeatElapsed >= n.cfg.HeartbeatTicks {
			n.heartbeatElapsed = 0
			n.broadcastAppend()
		}
		return
	}
	n.electionElapsed++
	if n.electionElapsed >= n.electionTimeout {
		n.campaign()
	}
}

// livro:fim tick

// livro:inicio campaign

// campaign inicia uma eleição: novo termo, voto em si mesmo e um pedido
// de voto a cada outro membro.
func (n *Node) campaign() {
	n.becomeCandidate()
	if n.quorum() == 1 {
		n.becomeLeader()
		return
	}
	for _, p := range n.others() {
		n.send(Message{
			Type:      MsgVote,
			To:        p,
			LastIndex: n.lastIndex(),
			LastTerm:  n.lastTerm(),
		})
	}
}

// livro:fim campaign

// livro:inicio papeis

func (n *Node) becomeFollower(term Term, leader NodeID) {
	if term > n.term {
		n.term, n.votedFor = term, 0
	}
	n.state, n.leader = Follower, leader
	n.resetElectionTimer()
}

func (n *Node) becomeCandidate() {
	n.state, n.leader = Candidate, 0
	n.term++
	n.votedFor = n.cfg.ID
	n.votes = map[NodeID]bool{n.cfg.ID: true}
	n.resetElectionTimer()
}

func (n *Node) becomeLeader() {
	n.state, n.leader = Leader, n.cfg.ID
	n.heartbeatElapsed = 0
	n.next, n.match = map[NodeID]Index{}, map[NodeID]Index{}
	for _, p := range n.cfg.Peers {
		n.next[p], n.match[p] = n.lastIndex()+1, 0
	}
	n.match[n.cfg.ID] = n.lastIndex()
	n.broadcastAppend()
}

// livro:fim papeis

func (n *Node) resetElectionTimer() {
	n.electionElapsed = 0
	n.electionTimeout = n.sortearTimeout()
}

// livro:inicio votar

// handleVote decide o voto. Um voto por termo: quem já votou em outro
// candidato neste termo nega.
func (n *Node) handleVote(m Message) {
	conceder := n.votedFor == 0 || n.votedFor == m.From
	if conceder {
		n.votedFor = m.From
		n.resetElectionTimer()
	}
	n.send(Message{Type: MsgVoteResp, To: m.From, Granted: conceder})
}

// handleVoteResp conta os votos; maioria vira liderança.
func (n *Node) handleVoteResp(m Message) {
	if n.state != Candidate {
		return
	}
	n.votes[m.From] = m.Granted
	aFavor := 0
	for _, g := range n.votes {
		if g {
			aFavor++
		}
	}
	if aFavor >= n.quorum() {
		n.becomeLeader()
	}
}

// livro:fim votar
