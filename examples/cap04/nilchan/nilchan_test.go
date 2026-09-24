package nilchan

import "testing"

func fonte(vs ...int) <-chan int {
	c := make(chan int)
	go func() {
		defer close(c)
		for _, v := range vs {
			c <- v
		}
	}()
	return c
}

func TestMesclarTerminaQuandoOsDoisFecham(t *testing.T) {
	soma, n := 0, 0
	for v := range Mesclar(fonte(1, 2, 3), fonte(10)) {
		soma += v
		n++
	}
	if soma != 16 || n != 4 {
		t.Fatalf("soma=%d n=%d", soma, n)
	}
}
