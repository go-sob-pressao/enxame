package sqlite_test

import (
	"path/filepath"
	"testing"

	"github.com/go-sob-pressao/enxame/internal/store/sqlite"
	"github.com/go-sob-pressao/enxame/internal/store/storetest"
)

func TestContrato(t *testing.T) {
	storetest.Run(t, func(t *testing.T) storetest.Store {
		s, err := sqlite.Open(
			t.Context(),
			filepath.Join(t.TempDir(), "enxame.db"),
		)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = s.Close() })
		return s
	})
}
