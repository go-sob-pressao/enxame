package worker

import (
	"context"
	"log/slog"
	"time"
)

// livro:inicio supervisionar

// Supervisionar roda f e, se ela devolver um erro antes de ctx
// terminar, roda de novo, com uma espera que dobra a cada falha
// seguida, de 100 ms até 10 s. Uma volta que durou mais de um minuto
// zera a espera. Um erro do banco derruba a volta, não o nó: o nó
// continua batendo, renovando e servindo, e o componente volta quando o
// banco voltar.
func Supervisionar(
	ctx context.Context,
	log *slog.Logger,
	nome string,
	f func(context.Context) error,
) error {
	const primeira, teto = 100 * time.Millisecond, 10 * time.Second
	espera := primeira
	for {
		inicio := time.Now()
		err := f(ctx)
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if time.Since(inicio) > time.Minute {
			espera = primeira
		}
		log.WarnContext(ctx, nome+": reiniciando",
			slog.Any("erro", err), slog.Duration("espera", espera))
		select {
		case <-time.After(espera):
		case <-ctx.Done():
			return ctx.Err()
		}
		espera = min(2*espera, teto)
	}
}

// livro:fim supervisionar
