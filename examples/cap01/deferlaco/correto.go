//go:build !defeito

package main

// livro:inicio defer-laco-correto

// Cada iteração vira uma função: o defer fecha ao fim de cada uma.
func processarTodos(p *Pool, n int) {
	for range n {
		processarUm(p)
	}
}

func processarUm(p *Pool) {
	r := p.Abrir()
	defer r.Close()
	processar(r)
}

// livro:fim defer-laco-correto

func processar(*Recurso) {}
