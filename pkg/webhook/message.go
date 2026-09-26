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

// Os jobs da entrega: o fan-out distribui a mensagem, criando um job
// de entrega por endpoint inscrito. Os dois vão para a mesma fila.
const (
	FanoutKind  = "webhook.fanout"
	DeliverKind = "webhook.deliver"
	FanoutQueue = "webhook"
)
