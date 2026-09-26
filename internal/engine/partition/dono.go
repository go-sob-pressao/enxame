package partition

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/go-sob-pressao/enxame/internal/store"
)

// Dono disputa uma partição e, enquanto a tiver, roda o trabalho dela.
type Dono struct {
	Lease    Lease
	Particao int
	Log      *slog.Logger
}

// livro:inicio dono

// Run tenta adquirir a partição a cada terço da duração do lease. Com a
// posse, roda trabalho com o token, e renova no mesmo ritmo. A posse
// termina quando a renovação é recusada — outro nó tem um token maior
// — ou quando o trabalho devolve store.ErrCercado; nos dois casos, o
// trabalho é cancelado e o Dono volta a disputar. Run não depende de
// perceber a perda a tempo: um processo pausado não percebe nada, e é
// para isso que o trabalho usa a cerca em toda transação.
func (d Dono) Run(
	ctx context.Context,
	trabalho func(ctx context.Context, token int64) error,
) error {
	ritmo := d.Lease.Duracao / 3
	for {
		token, ok, err := d.Lease.Adquirir(ctx, d.Particao)
		if err == nil && ok {
			err = d.servir(ctx, token, ritmo, trabalho)
		}
		if err != nil && ctx.Err() == nil {
			d.Log.WarnContext(ctx, "partição: posse encerrada",
				slog.Int("particao", d.Particao), slog.Any("erro", err))
		}
		select {
		case <-ctx.Done():
			return nil
		case <-time.After(ritmo):
		}
	}
}

// servir roda o trabalho enquanto a posse durar.
func (d Dono) servir(
	ctx context.Context,
	token int64,
	ritmo time.Duration,
	trabalho func(context.Context, int64) error,
) error {
	d.Log.InfoContext(ctx, "partição adquirida",
		slog.Int("particao", d.Particao), slog.Int64("range_id", token))
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	fim := make(chan error, 1)
	go func() { fim <- trabalho(ctx, token) }()
	t := time.NewTicker(ritmo)
	defer t.Stop()
	for {
		select {
		case err := <-fim:
			if errors.Is(err, store.ErrCercado) || ctx.Err() == nil {
				return err
			}
			return nil
		case <-t.C:
			ok, err := d.Lease.Renovar(ctx, d.Particao, token)
			if err != nil || !ok {
				cancel()
				<-fim
				if err == nil {
					err = errors.New("renovação recusada: outro dono")
				}
				return err
			}
		case <-ctx.Done():
			<-fim
			// Soltar com um contexto novo: o de Run já acabou.
			_ = d.Lease.Soltar(context.WithoutCancel(ctx), d.Particao,
				token)
			return nil
		}
	}
}

// livro:fim dono
