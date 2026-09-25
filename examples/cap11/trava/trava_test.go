//go:build !defeito

package trava

import (
	"testing"
	"testing/synctest"
	"time"
)

func TestLeituraDuranteRenovacao(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		var c Cache
		go c.Renovar(func() string { return "novo" })
		synctest.Wait() // a renovação está no Sleep, sem a trava
		if v := c.Valor(); v != "" {
			t.Fatalf("antes da renovação: %q", v)
		}
		synctest.Sleep(2 * time.Second)
		if v := c.Valor(); v != "novo" {
			t.Fatalf("depois da renovação: %q", v)
		}
	})
}
