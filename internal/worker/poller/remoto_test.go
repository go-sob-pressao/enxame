package poller

import (
	"context"
	"errors"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/go-sob-pressao/enxame/internal/core/job"
)

func TestRepetir(t *testing.T) {
	perdida := errors.Join(job.ErrInvalidTransition,
		status.Error(codes.FailedPrecondition, "perdida"))
	casos := []struct {
		nome      string
		erros     []error
		chamadas  int
		devolveIs error
	}{
		{"sucesso", []error{nil}, 1, nil},
		{"rede volta", []error{status.Error(codes.Unavailable, "x"),
			nil}, 2, nil},
		{"rede não volta", []error{
			status.Error(codes.Unavailable, "x"),
			status.Error(codes.Unavailable, "x"),
			status.Error(codes.Unavailable, "x")}, 3, nil},
		{"tentativa perdida", []error{perdida}, 1,
			job.ErrInvalidTransition},
	}
	for _, c := range casos {
		t.Run(c.nome, func(t *testing.T) {
			n := 0
			err := repetir(context.Background(), func() error {
				n++
				return c.erros[n-1]
			})
			if n != c.chamadas {
				t.Fatalf("%d chamadas; esperadas %d", n, c.chamadas)
			}
			if c.devolveIs == nil && err != nil ||
				c.devolveIs != nil && !errors.Is(err, c.devolveIs) {
				t.Fatalf("devolveu %v", err)
			}
		})
	}
}
