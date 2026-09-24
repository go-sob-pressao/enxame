// Package coordinator é o contrato; não pode conhecer a implementação.
package coordinator

import "exemplo.com/loja/internal/cluster/pgcoord"

// Padrao aponta, indevidamente, para a implementação.
const Padrao = pgcoord.Nome
