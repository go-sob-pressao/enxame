package job

import "context"

// Info descreve, dentro do handler, a tentativa em curso.
type Info struct {
	ID             string
	Kind           string
	Attempt        int
	MaxAttempts    int
	IdempotencyKey string
}

type chave struct{}

// WithInfo é usado pelo runtime que chama o handler.
func WithInfo(ctx context.Context, i Info) context.Context {
	return context.WithValue(ctx, chave{}, i)
}

// FromContext devolve a Info da tentativa em curso, se houver.
func FromContext(ctx context.Context) (Info, bool) {
	i, ok := ctx.Value(chave{}).(Info)
	return i, ok
}

// livro:inicio chave-handler

// IdempotencyKey devolve a chave de idempotência do trabalho em curso:
// a mesma em todas as tentativas de um job, ou em todos os replays de
// um passo de workflow. É a chave a mandar ao sistema externo — um
// gateway de pagamento, um provedor de e-mail — para que ele reconheça
// a repetição. Fora de um handler, devolve "".
func IdempotencyKey(ctx context.Context) string {
	i, _ := FromContext(ctx)
	return i.IdempotencyKey
}

// livro:fim chave-handler
