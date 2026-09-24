package flag

import "time"

// pararEmUmSegundo inicia o trabalho, pede parada e informa se parou.
func pararEmUmSegundo() bool {
	w := Novo()
	fim := make(chan struct{})
	go func() {
		n := 0
		w.Trabalhar(&n)
		close(fim)
	}()
	<-time.After(10 * time.Millisecond)
	w.Parar()
	select {
	case <-fim:
		return true
	case <-time.After(time.Second):
		return false
	}
}
