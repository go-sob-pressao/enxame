package postgres

import (
	"context"
	"embed"
	"fmt"
	"hash/fnv"
	"io/fs"
	"slices"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

//go:embed migrations/*.sql
var migracoes embed.FS

// Migracao é um par up/down, numerado.
type Migracao struct {
	Versao   int
	Nome     string
	Up, Down string
}

// Migracoes devolve as migrações do Enxame, em ordem.
func Migracoes() ([]Migracao, error) {
	sub, err := fs.Sub(migracoes, "migrations")
	if err != nil {
		return nil, err
	}
	return Carregar(sub)
}

// Carregar lê os pares NNNN_nome.up.sql e NNNN_nome.down.sql de fsys.
// Uma migração sem o down correspondente é recusada: toda migração tem
// volta.
func Carregar(fsys fs.FS) ([]Migracao, error) {
	nomes, err := fs.Glob(fsys, "*.up.sql")
	if err != nil {
		return nil, err
	}
	slices.Sort(nomes)
	var ms []Migracao
	for _, n := range nomes {
		base := strings.TrimSuffix(n, ".up.sql")
		num, _, _ := strings.Cut(base, "_")
		v, err := strconv.Atoi(num)
		if err != nil {
			return nil, fmt.Errorf("migração %s: número inválido", n)
		}
		up, err := fs.ReadFile(fsys, n)
		if err != nil {
			return nil, err
		}
		down, err := fs.ReadFile(fsys, base+".down.sql")
		if err != nil {
			return nil, fmt.Errorf("migração %s sem volta: %w",
				base, err)
		}
		ms = append(ms, Migracao{Versao: v, Nome: base,
			Up: string(up), Down: string(down)})
	}
	return ms, nil
}

// Migrate leva o esquema do Enxame à última versão.
func Migrate(ctx context.Context, pool *pgxpool.Pool) error {
	ms, err := Migracoes()
	if err != nil {
		return err
	}
	return Migrar(ctx, pool, "enxame_schema", ms, -1)
}

// livro:inicio migrar

// Migrar leva o esquema à versão alvo (-1: a última), aplicando os ups
// que faltam ou os downs que sobram, um por transação. A versão
// corrente fica gravada na tabela de controle, na mesma transação da
// migração: uma migração que falha no meio não conta como aplicada. Um
// advisory lock impede dois processos de migrarem ao mesmo tempo.
func Migrar(
	ctx context.Context,
	pool *pgxpool.Pool,
	tabela string,
	ms []Migracao,
	alvo int,
) error {
	conn, err := pool.Acquire(ctx)
	if err != nil {
		return err
	}
	defer conn.Release()
	h := fnv.New64a()
	h.Write([]byte(tabela))
	chave := int64(h.Sum64())
	if _, err := conn.Exec(ctx,
		`SELECT pg_advisory_lock($1)`, chave); err != nil {
		return err
	}
	defer conn.Exec(context.WithoutCancel(ctx), //nolint:errcheck
		`SELECT pg_advisory_unlock($1)`, chave)
	_, err = conn.Exec(ctx, `CREATE TABLE IF NOT EXISTS `+tabela+
		` (versao INT PRIMARY KEY, nome TEXT NOT NULL,
		   aplicada_em TIMESTAMPTZ NOT NULL DEFAULT now())`)
	if err != nil {
		return err
	}
	var atual int
	if err := conn.QueryRow(ctx, `SELECT coalesce(max(versao), 0)
		FROM `+tabela).Scan(&atual); err != nil {
		return err
	}
	if alvo < 0 && len(ms) > 0 {
		alvo = ms[len(ms)-1].Versao
	}
	for _, m := range ms { // subir
		if m.Versao <= atual || m.Versao > alvo {
			continue
		}
		if err := passo(ctx, conn.Conn(), m.Up, m,
			`INSERT INTO `+tabela+` (versao, nome) VALUES ($1, $2)`,
		); err != nil {
			return fmt.Errorf("up %s: %w", m.Nome, err)
		}
	}
	for _, m := range slices.Backward(ms) { // descer
		if m.Versao > atual || m.Versao <= alvo {
			continue
		}
		if err := passo(ctx, conn.Conn(), m.Down, m,
			`DELETE FROM `+tabela+` WHERE versao = $1 AND nome = $2`,
		); err != nil {
			return fmt.Errorf("down %s: %w", m.Nome, err)
		}
	}
	return nil
}

// livro:fim migrar

func passo(
	ctx context.Context,
	conn *pgx.Conn,
	sql string,
	m Migracao,
	controle string,
) error {
	return pgx.BeginFunc(ctx, conn, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, sql); err != nil {
			return err
		}
		_, err := tx.Exec(ctx, controle, m.Versao, m.Nome)
		return err
	})
}
