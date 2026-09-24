//go:build defeito

package main

// livro:inicio defer-laco-defeito

// processarTodos parece fechar cada recurso. Mas defer executa quando a
// FUNÇÃO retorna, não quando a iteração termina: os n ficam abertos
// juntos.
func processarTodos(p *Pool, n int) {
	for range n {
		r := p.Abrir()
		defer r.Close()
		processar(r)
	}
}

// livro:fim defer-laco-defeito

func processar(*Recurso) {}
