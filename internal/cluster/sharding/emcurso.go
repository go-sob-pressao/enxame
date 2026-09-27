package sharding

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

// EmCursoNoBanco conta as tentativas em execução pela tabela job.
func EmCursoNoBanco(db *pgxpool.Pool) EmCurso {
	return func(ctx context.Context, ps []int) (map[int]int, error) {
		qs := make([]int32, len(ps))
		for i, p := range ps {
			qs[i] = int32(p) //nolint:gosec // < 512
		}
		rows, err := db.Query(ctx, `SELECT partition_id, count(*)
			FROM job WHERE partition_id = ANY($1)
			  AND state = 'running'
			GROUP BY partition_id`, qs)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		n := map[int]int{}
		for rows.Next() {
			var p, c int
			if err := rows.Scan(&p, &c); err != nil {
				return nil, err
			}
			n[p] = c
		}
		return n, rows.Err()
	}
}
