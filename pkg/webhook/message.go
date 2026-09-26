package webhook

// Message é um evento que a aplicação publica para os endpoints
// inscritos no tipo dele. A entrega é do Capítulo 20; o que existe
// desde o Capítulo 15 é a garantia de que a mensagem é gravada na mesma
// transação dos dados de negócio.
type Message struct {
	EventType      string
	Payload        any    // serializado em JSON
	IdempotencyKey string // opcional: a mesma chave não grava de novo
}

// Os nomes do job que distribui a mensagem aos endpoints.
const (
	FanoutKind  = "webhook.fanout"
	FanoutQueue = "webhook"
)
