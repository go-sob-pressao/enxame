package worker_test

import (
	"context"
	"errors"
	"log/slog"
	"testing"
	"testing/synctest"

	"github.com/go-sob-pressao/enxame/internal/worker"
)

func TestSupervisionarReiniciaAteOFim(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		ctx, parar := context.WithCancel(t.Context())
		voltas := 0
		err := worker.Supervisionar(ctx, slog.New(slog.DiscardHandler), "x",
			func(context.Context) error {
				voltas++
				if voltas == 4 {
					parar()
				}
				return errors.New("banco caiu")
			})
		if !errors.Is(err, context.Canceled) || voltas != 4 {
			t.Fatalf("voltas %d, erro %v", voltas, err)
		}
	})
}
