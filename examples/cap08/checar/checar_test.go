package checar

import "sync"

// obterEmParalelo pede a mesma conexão de 64 goroutines ao mesmo tempo.
func obterEmParalelo() int64 {
	var c Cache
	var comeco, wg sync.WaitGroup
	comeco.Add(1)
	for range 64 {
		wg.Go(func() {
			comeco.Wait()
			c.Obter("https://cliente.exemplo/webhook")
		})
	}
	comeco.Done()
	wg.Wait()
	return Abertas()
}
