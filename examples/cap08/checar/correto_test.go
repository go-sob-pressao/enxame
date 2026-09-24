//go:build !defeito

package checar

import "testing"

func TestUmaConexaoPorEndpoint(t *testing.T) {
	for range 100 {
		if n := obterEmParalelo(); n != 1 {
			t.Fatalf("%d conexões abertas para o mesmo endpoint", n)
		}
	}
}
