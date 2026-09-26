package main

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/go-sob-pressao/enxame/internal/engine/partition"
	"github.com/go-sob-pressao/enxame/internal/store"
	"github.com/go-sob-pressao/enxame/internal/store/postgres"
)

// no é um processo que, enquanto for dono da partição 0, acrescenta
// lançamentos ao extrato, numerados em sequência. O próximo número vive
// na memória do dono — o estado exato por partição que justifica ter
// um dono (Cap. 22).
type no struct {
	db    *pgxpool.Pool
	lease partition.Lease
	cerca bool // confere o token em cada escrita
	log   func(string, ...any)
}

// livro:inicio lease-ingenuo

// servir escreve um lançamento a cada 100 ms enquanto acredita ser o
// dono, e renova a posse a cada terço do lease. Entre uma renovação e
// outra, a crença não é conferida com ninguém: o nó escreve porque, da
// última vez que perguntou, era o dono.
func (n *no) servir(ctx context.Context, token int64) error {
	seq, err := n.ultimo(ctx)
	if err != nil {
		return err
	}
	renovado := time.Now()
	for ctx.Err() == nil {
		seq++
		if err := n.escrever(ctx, seq, token); err != nil {
			return err
		}
		if time.Since(renovado) > n.lease.Duracao/3 {
			ok, err := n.lease.Renovar(ctx, 0, token)
			if err != nil || !ok {
				return fmt.Errorf("perdi a posse (token %d)", token)
			}
			renovado = time.Now()
		}
		time.Sleep(100 * time.Millisecond)
	}
	return nil
}

// livro:fim lease-ingenuo

// livro:inicio escrever

// escrever grava o lançamento seq. Com a cerca, a transação começa
// conferindo o token com FOR SHARE; sem ela, grava e pronto.
func (n *no) escrever(ctx context.Context, seq int, token int64) error {
	return pgx.BeginFunc(ctx, n.db, func(tx pgx.Tx) error {
		if n.cerca {
			err := postgres.ConferirCerca(ctx, tx,
				postgres.Cerca{Particao: 0, RangeID: token})
			if err != nil {
				return err
			}
		}
		_, err := tx.Exec(ctx, `INSERT INTO extrato (seq, no, token)
			VALUES ($1, $2, $3)`, seq, n.lease.No, token)
		return err
	})
}

// livro:fim escrever

func (n *no) ultimo(ctx context.Context) (int, error) {
	var seq int
	err := n.db.QueryRow(ctx, `SELECT coalesce(max(seq), 0)
		FROM extrato`).Scan(&seq)
	return seq, err
}

// rodar disputa a partição e a serve, de novo e de novo.
func (n *no) rodar(ctx context.Context) {
	for ctx.Err() == nil {
		token, ok, err := n.lease.Adquirir(ctx, 0)
		if err == nil && ok {
			n.log("adquiri a partição 0 com o token %d", token)
			err = n.servir(ctx, token)
			switch {
			case errors.Is(err, store.ErrCercado):
				n.log("escrita recusada: %v", err)
			case err != nil:
				n.log("%v", err)
			}
		}
		time.Sleep(200 * time.Millisecond)
	}
}
