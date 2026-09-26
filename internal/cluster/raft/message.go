package raft

// MessageType identifica a mensagem trocada entre nós.
type MessageType uint8

// Tipos de mensagem.
const (
	MsgVote     MessageType = iota + 1 // RequestVote
	MsgVoteResp                        // resposta ao RequestVote
	MsgApp                             // AppendEntries (e heartbeat)
	MsgAppResp                         // resposta ao AppendEntries
)

// Message é uma mensagem entre dois nós. Só os campos do tipo
// correspondente são preenchidos.
type Message struct {
	Type MessageType
	From NodeID
	To   NodeID
	Term Term

	Granted bool // MsgVoteResp
}
