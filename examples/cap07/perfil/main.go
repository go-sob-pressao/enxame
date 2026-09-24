// Command perfil: o perfil goroutineleak (Go 1.27) num serviço em
// execução.
//
//	go run ./examples/cap07/perfil &
//	curl -s 'localhost:6060/debug/pprof/goroutineleak?debug=1' | head -30
//	curl -s 'localhost:6060/debug/pprof/goroutine?debug=1'     | head -30
//
// O perfil de goroutines lista TODAS; o goroutineleak lista as
// bloqueadas numa primitiva que nenhuma goroutine viva alcança mais —
// as que o GC consegue provar que nunca vão acordar.
package main

import (
	"fmt"
	"log/slog"
	"net/http"
	_ "net/http/pprof" //nolint:gosec // exemplo local de diagnóstico
	"os"
	"time"
)

// livro:inicio perfil-vazamento

// consultar vaza uma goroutine por chamada: o canal só é alcançável
// pela goroutine presa, então o GC prova que ela nunca vai acordar.
func consultar() int {
	c := make(chan int)
	go func() { c <- 42 }()
	select {
	case v := <-c:
		return v
	default:
		return 0
	}
}

// global é alcançável pelo pacote inteiro: uma goroutine presa nele NÃO
// aparece no goroutineleak — o GC não pode provar que ninguém enviará.
var global = make(chan int)

func vazarEmGlobal() { go func() { <-global }() }

// livro:fim perfil-vazamento

func main() {
	log := slog.New(slog.NewTextHandler(os.Stderr, nil))
	for range 100 {
		consultar()
		vazarEmGlobal()
	}
	fmt.Println(
		"100 vazamentos locais e 100 em variável global; perfis em localhost:6060/debug/pprof/",
	)
	srv := &http.Server{
		Addr:              "localhost:6060",
		ReadHeaderTimeout: 5 * time.Second,
	}
	if err := srv.ListenAndServe(); err != nil {
		log.Error("pprof", slog.Any("erro", err))
	}
}
