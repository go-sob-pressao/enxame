// Package pipeline — estágios canceláveis: gerar → filtrar → somar.
package pipeline

import "context"

// livro:inicio pipeline

// Gerar emite de 1 a n. Cada estágio é dono do canal que cria e o fecha
// quando termina; todo envio escuta o cancelamento.
func Gerar(ctx context.Context, n int) <-chan int {
	saida := make(chan int)
	go func() {
		defer close(saida)
		for i := 1; i <= n; i++ {
			select {
			case saida <- i:
			case <-ctx.Done():
				return
			}
		}
	}()
	return saida
}

// Filtrar deixa passar os valores que satisfazem ok.
func Filtrar(
	ctx context.Context,
	entrada <-chan int,
	ok func(int) bool,
) <-chan int {
	saida := make(chan int)
	go func() {
		defer close(saida)
		for v := range entrada {
			if !ok(v) {
				continue
			}
			select {
			case saida <- v:
			case <-ctx.Done():
				return
			}
		}
	}()
	return saida
}

// Somar consome até o fim ou até o cancelamento.
func Somar(ctx context.Context, entrada <-chan int) (int, error) {
	total := 0
	for {
		select {
		case v, ok := <-entrada:
			if !ok {
				return total, nil
			}
			total += v
		case <-ctx.Done():
			return total, ctx.Err()
		}
	}
}

// livro:fim pipeline
