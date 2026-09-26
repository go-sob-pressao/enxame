package http

import (
	"math"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/go-sob-pressao/enxame/internal/core/policy"
)

// livro:inicio limitar

// Limitar dá a cada namespace um balde de fichas: taxa requisições por
// segundo sustentadas, rajada de uma vez. Sem ficha, 429 com
// Retry-After — o cliente fica sabendo quando voltar, em vez de
// adivinhar. Vem depois do Autenticar: o namespace é quem paga.
func Limitar(taxa, rajada float64) Middleware {
	var mu sync.Mutex
	baldes := map[string]*policy.TokenBucket{}
	return func(prox http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter,
			r *http.Request) {
			ns := namespace(r.Context())
			mu.Lock()
			b, ok := baldes[ns]
			if !ok {
				b = &policy.TokenBucket{Taxa: taxa, Rajada: rajada}
				baldes[ns] = b
			}
			pode, espera := b.Tomar(agora())
			mu.Unlock()
			if !pode {
				recusar(w, http.StatusTooManyRequests, espera,
					Erro{"rate_limited", "taxa do namespace excedida"})
				return
			}
			prox.ServeHTTP(w, r)
		})
	}
}

// livro:fim limitar

// livro:inicio descartar

// Descartar limita as requisições em curso a n. A que chega com
// todas as vagas ocupadas é recusada na hora com 503, sem entrar em
// fila nenhuma: uma resposta rápida de "agora não" custa quase nada, e
// a alternativa é esperar por uma conexão do banco que não vai vagar a
// tempo. Quem está dentro continua com a latência de sempre.
func Descartar(n int) Middleware {
	vagas := make(chan struct{}, n)
	return func(prox http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter,
			r *http.Request) {
			select {
			case vagas <- struct{}{}:
				defer func() { <-vagas }()
				prox.ServeHTTP(w, r)
			default:
				recusar(w, http.StatusServiceUnavailable, time.Second,
					Erro{"overloaded", "servidor sobrecarregado"})
			}
		})
	}
}

// livro:fim descartar

// recusar responde com Retry-After em segundos inteiros, arredondado
// para cima: esperar um pouco a mais custa menos que voltar cedo.
func recusar(w http.ResponseWriter, status int, espera time.Duration,
	e Erro) {
	s := max(1, int(math.Ceil(espera.Seconds())))
	w.Header().Set("Retry-After", strconv.Itoa(s))
	escrever(w, status, e)
}
