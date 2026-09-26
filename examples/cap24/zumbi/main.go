// Command zumbi é o Experimento 24.1 e o 24.2: dois processos disputam
// a partição 0 com um lease; o dono é pausado com SIGSTOP, o outro
// assume, e o pausado volta com SIGCONT (Capítulo 24).
//
//	go run ./examples/cap24/zumbi preparar
//	go run ./examples/cap24/zumbi -no a [-cerca] &
//	kill -STOP %1; go run ./examples/cap24/zumbi -no b [-cerca] &
//	kill -CONT %1
//	go run ./examples/cap24/zumbi conferir
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/go-sob-pressao/enxame/internal/engine/partition"
	"github.com/go-sob-pressao/enxame/internal/store/postgres"
)

func main() {
	dsn := flag.String("dsn", os.Getenv("ENXAME_DB_DSN"), "PostgreSQL")
	nome := flag.String("no", "a", "nome deste nó")
	cerca := flag.Bool("cerca", false, "conferir o token ao escrever")
	duracao := flag.Duration("lease", time.Second, "duração do lease")
	flag.Parse()
	ctx, parar := signal.NotifyContext(context.Background(),
		syscall.SIGTERM, os.Interrupt)
	defer parar()
	db, err := pgxpool.New(ctx, *dsn)
	if err != nil {
		sair(err)
	}
	defer db.Close()
	switch flag.Arg(0) {
	case "preparar":
		sair(preparar(ctx, db))
	case "conferir":
		sair(conferir(ctx, db))
	}
	n := &no{db: db, cerca: *cerca,
		lease: partition.Lease{DB: db, No: *nome, Duracao: *duracao},
		log: func(f string, a ...any) {
			fmt.Printf("%s %s: %s\n", time.Now().Format("15:04:05.000"),
				*nome, fmt.Sprintf(f, a...))
		}}
	n.rodar(ctx)
}

func preparar(ctx context.Context, db *pgxpool.Pool) error {
	if err := postgres.Migrate(ctx, db); err != nil {
		return err
	}
	_, err := db.Exec(ctx, `DROP TABLE IF EXISTS extrato;
		CREATE TABLE extrato (seq INT, no TEXT, token BIGINT,
			em TIMESTAMPTZ DEFAULT clock_timestamp());
		UPDATE partition_lease SET owner = NULL,
			lease_expires_at = NULL WHERE partition_id = 0`)
	return err
}

// conferir mostra os números de lançamento usados mais de uma vez.
func conferir(ctx context.Context, db *pgxpool.Pool) error {
	rows, err := db.Query(ctx, `SELECT seq, string_agg(no || ' (token '
		|| token || ')', ', ' ORDER BY em) FROM extrato
		GROUP BY seq HAVING count(*) > 1 ORDER BY seq`)
	if err != nil {
		return err
	}
	defer rows.Close()
	var total, dup int
	_ = db.QueryRow(ctx, `SELECT count(*) FROM extrato`).Scan(&total)
	for rows.Next() {
		var seq int
		var quem string
		if err := rows.Scan(&seq, &quem); err != nil {
			return err
		}
		fmt.Printf("lançamento %d gravado por %s\n", seq, quem)
		dup++
	}
	fmt.Printf("%d lançamentos; números usados duas vezes: %d\n",
		total, dup)
	return rows.Err()
}

func sair(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	os.Exit(0)
}
