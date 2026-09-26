package raft

// MessageType identifica a mensagem trocada entre nós.
type MessageType uint8

// Tipos de mensagem.
const (
	MsgVote     MessageType = iota + 1 // RequestVote
	MsgVoteResp                        // resposta ao RequestVote
	MsgApp                             // AppendEntries (e heartbeat)
	MsgAppResp                         // resposta ao AppendEntries
	MsgSnap                            // InstallSnapshot
)

// Message é uma mensagem entre dois nós. Só os campos do tipo
// correspondente são preenchidos.
type Message struct {
	Type MessageType
	From NodeID
	To   NodeID
	Term Term

	Granted bool // MsgVoteResp

	// MsgVote: o fim do log do candidato.
	LastIndex Index
	LastTerm  Term

	// MsgApp: a entrada que precede as novas, as novas e o commit do
	// líder.
	PrevIndex Index
	PrevTerm  Term
	Entries   []Entry
	Commit    Index

	// MsgAppResp: aceitou? até onde o log agora coincide com o do
	// líder; se recusou, uma dica de onde tentar de novo.
	Success bool
	Match   Index
	Hint    Index

	// MsgSnap: o snapshot do líder.
	Snapshot *Snapshot
}
