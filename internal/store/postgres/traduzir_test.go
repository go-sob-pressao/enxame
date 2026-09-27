package postgres

import (
	"errors"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"

	"github.com/go-sob-pressao/enxame/internal/store"
)

func TestTraduzirDiscoCheio(t *testing.T) {
	err := traduzir(&pgconn.PgError{Code: "53100",
		Message: "could not extend file"})
	if !errors.Is(err, store.ErrSemRecursos) {
		t.Fatalf("disco cheio virou %v", err)
	}
}
