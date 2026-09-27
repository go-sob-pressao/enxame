//go:build chaos

package chaos

import (
	"context"
	"fmt"
	"os"
	"testing"
)

var bin Binarios

func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "caos")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if bin, err = Compilar(context.Background(), dir); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	c := m.Run()
	_ = os.RemoveAll(dir)
	os.Exit(c)
}
