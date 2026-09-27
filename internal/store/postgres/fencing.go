package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/go-sob-pressao/enxame/internal/store"
)

// Cerca é o fencing token que o dono de uma partição carrega: o
// range_id que ele recebeu na aquisição (ADR-005).
type Cerca struct {
	Particao int
	RangeID  int64
}

// ComCerca devolve uma cópia do Store cujas transações do motor —
// reservar, decidir, promover, resgatar, disparar agendamentos, gravar
// passos — começam conferindo a cerca. Quem não é mais dono recebe
// store.ErrCercado, e nada do que a transação escreveu fica.
func (s *Store) ComCerca(c Cerca) *Store {
	n := *s
	n.cerca = &c
	return &n
}

// RelogioDoBanco devolve uma cópia do Store que usa o now() do banco,
// e não o instante que o chamador passa, na reserva, no batimento, na
// promoção e no resgate: um relógio só para todos os nós (Caps. 22 e
// 24). O instante injetado continua valendo nos testes do contrato.
func (s *Store) RelogioDoBanco() *Store {
	n := *s
	n.banco = true
	return &n
}

// livro:inicio fencing

// ConferirCerca é a primeira instrução de toda transação do dono. O
// FOR SHARE é o que a faz funcionar: enquanto esta transação estiver
// aberta, a aquisição de outro nó — um UPDATE na mesma linha — espera
// por ela. Tudo o que o dono antigo escreve fica antes da troca de
// dono; tudo o que ele tentar depois lê o range_id novo e é recusado.
func ConferirCerca(ctx context.Context, tx pgx.Tx, c Cerca) error {
	var atual int64
	err := tx.QueryRow(ctx, `SELECT range_id FROM partition_lease
		WHERE partition_id = $1 FOR SHARE`, c.Particao).Scan(&atual)
	if err != nil {
		return err
	}
	if atual != c.RangeID {
		return fmt.Errorf("%w: partição %d, range_id %d, o atual é %d",
			store.ErrCercado, c.Particao, c.RangeID, atual)
	}
	return nil
}

// transacao abre uma transação do motor, com a cerca conferida antes
// de qualquer outra coisa, se o Store carregar uma.
func (s *Store) transacao(
	ctx context.Context,
	fn func(pgx.Tx) error,
) error {
	return pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		if s.cerca != nil {
			if err := ConferirCerca(ctx, tx, *s.cerca); err != nil {
				return err
			}
		}
		return fn(tx)
	})
}

// livro:fim fencing

// instante devolve at, ou o now() do banco se o Store usa o relógio do
// banco.
func (s *Store) instante(ctx context.Context, at time.Time) (time.Time,
	error) {
	if !s.banco {
		return at, nil
	}
	var agora time.Time
	err := s.pool.QueryRow(ctx, `SELECT now()`).Scan(&agora)
	return agora, err
}

// Dono diz ao Store quais partições este nó tem, com o token de cada
// uma, e a quem avisar quando uma delas se revelar perdida.
type Dono struct {
	Tokens func() map[int]int64
	Perdeu func(particao int)
}

// ComDono devolve uma cópia do Store cuja reserva só pega jobs das
// partições do nó, conferindo o token da partição do job reservado. O
// modo biblioteca não usa: sem dono, a reserva pega de todas.
func (s *Store) ComDono(d Dono) *Store {
	n := *s
	n.dono = &d
	return &n
}

// particaoDaCerca restringe promoção e resgate à partição da cerca, se
// houver: cada dono cuida só das suas.
func (s *Store) particaoDaCerca() string {
	if s.cerca == nil {
		return ""
	}
	return fmt.Sprintf(" AND partition_id = %d", s.cerca.Particao)
}

// livro:inicio cercas

// conferirCercas confere, com um FOR SHARE só, os tokens de todas as
// partições do nó, e devolve as que continuam dele e as perdidas. A
// promoção e o resgate do dono agem só nas válidas, na mesma transação.
func conferirCercas(
	ctx context.Context,
	tx pgx.Tx,
	tokens map[int]int64,
) (validas []int32, perdidas []int, err error) {
	ps := make([]int32, 0, len(tokens))
	for p := range tokens {
		ps = append(ps, int32(p)) //nolint:gosec // < 512
	}
	rows, err := tx.Query(ctx, `SELECT partition_id, range_id
		FROM partition_lease WHERE partition_id = ANY($1)
		ORDER BY partition_id FOR SHARE`, ps)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var p int
		var r int64
		if err := rows.Scan(&p, &r); err != nil {
			return nil, nil, err
		}
		if r == tokens[p] {
			validas = append(validas, int32(p)) //nolint:gosec // < 512
		} else {
			perdidas = append(perdidas, p)
		}
	}
	return validas, perdidas, rows.Err()
}

// livro:fim cercas
