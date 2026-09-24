// Package nilchan — nil channel no select para desligar um caso.
package nilchan

// livro:inicio mesclar

// Mesclar junta dois canais até os DOIS fecharem. Quando um fecha, a
// variável dele vira nil: receber de nil bloqueia para sempre, então o
// select simplesmente deixa de escolher aquele caso.
func Mesclar(a, b <-chan int) <-chan int {
	saida := make(chan int)
	go func() {
		defer close(saida)
		for a != nil || b != nil {
			select {
			case v, ok := <-a:
				if !ok {
					a = nil
					continue
				}
				saida <- v
			case v, ok := <-b:
				if !ok {
					b = nil
					continue
				}
				saida <- v
			}
		}
	}()
	return saida
}

// livro:fim mesclar
