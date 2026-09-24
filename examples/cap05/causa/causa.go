// Package causa — cancelado ou expirado, e por que a diferença importa.
package causa

import (
	"context"
	"errors"
)

// ErrDesligamento é a causa registrada quando o processo está parando.
var ErrDesligamento = errors.New("desligamento do worker")

// Decisao é o que fazer com uma tentativa interrompida.
type Decisao string

// Decisões possíveis.
const (
	// não é culpa do job: tente de novo sem contar tentativa
	Repetir Decisao = "repetir"
	// estourou o prazo: conta como tentativa falha
	ContarFalha Decisao = "contar-falha"
	// cancelado por quem pediu: não repita
	Desistir Decisao = "desistir"
)

// livro:inicio classificar

// Classificar decide o destino de uma tentativa pelo erro e pela CAUSA.
// Canceled e DeadlineExceeded pedem decisões opostas: confundir os dois
// significa repetir o que foi cancelado ou perdoar o que travou.
func Classificar(ctx context.Context, err error) Decisao {
	switch {
	case errors.Is(err, context.DeadlineExceeded):
		return ContarFalha
	case errors.Is(err, context.Canceled) &&
		errors.Is(context.Cause(ctx), ErrDesligamento):
		return Repetir
	case errors.Is(err, context.Canceled):
		return Desistir
	default:
		return ContarFalha
	}
}

// livro:fim classificar
