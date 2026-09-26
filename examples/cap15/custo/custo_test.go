package custo_test

import (
	"sync/atomic"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/go-sob-pressao/enxame/pkg/enxame"
	"github.com/go-sob-pressao/enxame/test/testutil"
)

type cobrar struct {
	PedidoID int64 `json:"pedido_id"`
}

func (cobrar) Kind() string { return "cobrar-pedido" }

var proximo atomic.Int64

func preparar(b *testing.B) (*pgxpool.Pool, *enxame.Client) {
	b.Helper()
	db := testutil.Postgres(b)
	if _, err := db.Exec(b.Context(), `CREATE TABLE pedido
		(id BIGINT PRIMARY KEY, total INT NOT NULL)`); err != nil {
		b.Fatal(err)
	}
	return db, enxame.New(db, "loja")
}

// livro:inicio custo-commit

// BenchmarkUmaTransacao: o pedido e o job no mesmo COMMIT.
func BenchmarkUmaTransacao(b *testing.B) {
	db, c := preparar(b)
	ctx := b.Context()
	criar := func(id int64) func(pgx.Tx) error {
		return func(tx pgx.Tx) error {
			_, err := tx.Exec(ctx,
				`INSERT INTO pedido VALUES ($1, 100)`, id)
			if err == nil {
				_, err = c.InsertTx(ctx, tx, cobrar{id})
			}
			return err
		}
	}
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			id := proximo.Add(1)
			if err := pgx.BeginFunc(ctx, db, criar(id)); err != nil {
				b.Fatal(err)
			}
		}
	})
}

// BenchmarkDuasTransacoes: o pedido num COMMIT, o job em outro.
func BenchmarkDuasTransacoes(b *testing.B) {
	db, c := preparar(b)
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			id := proximo.Add(1)
			if _, err := db.Exec(b.Context(),
				`INSERT INTO pedido VALUES ($1, 100)`, id); err != nil {
				b.Fatal(err)
			}
			if _, err := c.Insert(b.Context(), cobrar{id}); err != nil {
				b.Fatal(err)
			}
		}
	})
}

// BenchmarkDualWrite: o pedido num COMMIT, a mensagem numa fila em
// memória — o limite inferior de qualquer broker, que ainda cobraria
// uma ida e volta pela rede.
func BenchmarkDualWrite(b *testing.B) {
	db, _ := preparar(b)
	fila := make(chan int64, 1024)
	go func() {
		for range fila { //nolint:revive // o consumidor só esvazia
		}
	}()
	defer close(fila)
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			id := proximo.Add(1)
			if _, err := db.Exec(b.Context(),
				`INSERT INTO pedido VALUES ($1, 100)`, id); err != nil {
				b.Fatal(err)
			}
			fila <- id
		}
	})
}

// livro:fim custo-commit
