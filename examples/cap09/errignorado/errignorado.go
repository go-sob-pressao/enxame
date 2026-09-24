// Package errignorado — err ignorado com _ (Anti-Pattern #11).
package errignorado

import "errors"

// ErrConflito simula a falha do COMMIT por conflito de serialização.
var ErrConflito = errors.New(
	"could not serialize access due to concurrent update",
)

// Tx simula uma transação.
type Tx struct {
	falharCommit bool
	gravado      *[]string
	pendente     []string
}

// Inserir registra uma linha na transação.
func (t *Tx) Inserir(
	linha string,
) {
	t.pendente = append(t.pendente, linha)
}

// Commit grava as linhas ou falha.
func (t *Tx) Commit() error {
	if t.falharCommit {
		return ErrConflito
	}
	*t.gravado = append(*t.gravado, t.pendente...)
	return nil
}
