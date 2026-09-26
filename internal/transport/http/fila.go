package http

import (
	"context"
	"net/http"
	"sync"
	"time"
)

// Fila limita a profundidade da fila de cada namespace.
type Fila struct {
	// Max é quantos jobs podem esperar execução no namespace; zero
	// desliga o limite.
	Max int
	// Validade é por quanto tempo uma contagem vale. Zero conta a cada
	// job — exato, e uma consulta a mais por inserção.
	Validade time.Duration

	mu        sync.Mutex
	contagens map[string]*contagem
}

type contagem struct {
	mu sync.Mutex
	n  int
	em time.Time
}

// livro:inicio fila-cheia

// filaCheia diz se a fila do namespace já tem Max jobs esperando. A
// contagem vale por Validade: o limite é aproximado, e pode passar do
// Max pelo que entra nesse intervalo — o preço de não contar a cada
// inserção. Quem chega durante uma contagem espera por ela, em vez de
// disparar outra.
func (a *API) filaCheia(ctx context.Context, ns string) (bool, error) {
	if a.Fila.Max <= 0 {
		return false, nil
	}
	c := a.Fila.contagem(ns)
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.em.IsZero() || agora().Sub(c.em) >= a.Fila.Validade {
		n, err := a.Store.Esperando(ctx, ns)
		if err != nil {
			return false, err
		}
		c.n, c.em = n, agora()
	}
	return c.n >= a.Fila.Max, nil
}

// livro:fim fila-cheia

func (f *Fila) contagem(ns string) *contagem {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.contagens == nil {
		f.contagens = map[string]*contagem{}
	}
	c, ok := f.contagens[ns]
	if !ok {
		c = &contagem{}
		f.contagens[ns] = c
	}
	return c
}

// recusarFilaCheia responde 429: a fila é do namespace, e o excesso
// também. Esperar a fila baixar leva mais que um instante.
func recusarFilaCheia(w http.ResponseWriter) {
	recusar(w, http.StatusTooManyRequests, 5*time.Second,
		Erro{"queue_full", "fila do namespace cheia"})
}
