// Package vazamentos — as quatro formas clássicas de vazar uma
// goroutine.
package vazamentos

import (
	"context"
	"time"
)

// livro:inicio vaza-envio

// PrimeiroResultado consulta todas as réplicas e devolve a primeira
// resposta. As outras goroutines ficam presas no envio: ninguém mais lê
// o canal.
func PrimeiroResultado(replicas []func() int) int {
	c := make(chan int)
	for _, r := range replicas {
		go func() { c <- r() }()
	}
	return <-c
}

// livro:fim vaza-envio

// livro:inicio vaza-recepcao

// Consumir processa o que chegar em c. Se quem produz esquece de fechar
// c, esta goroutine espera para sempre por um valor que nunca virá.
func Consumir(c <-chan int, processar func(int)) {
	go func() {
		for v := range c {
			processar(v)
		}
	}()
}

// livro:fim vaza-recepcao

// livro:inicio vaza-contexto

// Vigiar verifica a saúde a cada intervalo. Recebe um contexto e o
// ignora: quem chamou cancela e acha que parou tudo.
func Vigiar(_ context.Context, verificar func()) {
	go func() {
		for {
			verificar()
			<-time.After(10 * time.Millisecond)
		}
	}()
}

// livro:fim vaza-contexto

// livro:inicio vaza-ticker

// Publicar envia métricas a cada tique. O laço só termina quando o
// ticker para — e ninguém o para.
func Publicar(enviar func()) {
	t := time.NewTicker(10 * time.Millisecond)
	go func() {
		for range t.C {
			enviar()
		}
	}()
}

// livro:fim vaza-ticker
