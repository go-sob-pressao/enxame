package runner

import (
	"fmt"
	"time"
)

// livro:inicio repetir-em

// RepetirEm é o erro com que um handler pede que a próxima tentativa
// não aconteça antes de Depois — o Retry-After de quem respondeu 429,
// ou a espera do próprio limite de taxa. O pool usa o maior entre esse
// pedido e o backoff: o backoff protege o outro lado de todos; o
// pedido, de quem sabe da própria capacidade.
type RepetirEm struct {
	Depois time.Duration
	Err    error
}

func (e *RepetirEm) Error() string {
	return fmt.Sprintf("%v (repetir depois de %v)", e.Err, e.Depois)
}

func (e *RepetirEm) Unwrap() error { return e.Err }

// livro:fim repetir-em
