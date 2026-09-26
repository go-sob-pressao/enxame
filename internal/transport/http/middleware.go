package http

import (
	"context"
	"crypto/subtle"
	"log/slog"
	"net/http"
	"slices"
	"strings"
	"time"
)

// Middleware embrulha um handler com um comportamento transversal.
type Middleware func(http.Handler) http.Handler

// livro:inicio middleware

// Encadear aplica os middlewares na ordem em que aparecem: o primeiro
// é o mais externo, o primeiro a ver a requisição e o último a ver a
// resposta. Nenhum framework: é só composição de funções.
func Encadear(h http.Handler, ms ...Middleware) http.Handler {
	for _, m := range slices.Backward(ms) {
		h = m(h)
	}
	return h
}

// Recuperar transforma o pânico de um handler em 500, em vez de
// derrubar a conexão sem resposta.
func (a *API) Recuperar(prox http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter,
		r *http.Request) {
		defer a.recuperar(w, r)
		prox.ServeHTTP(w, r)
	})
}

func (a *API) recuperar(w http.ResponseWriter, r *http.Request) {
	if v := recover(); v != nil {
		a.Log.ErrorContext(r.Context(), "pânico no handler",
			slog.Any("valor", v), slog.String("rota", r.Pattern))
		escrever(w, http.StatusInternalServerError,
			Erro{"internal", "erro interno"})
	}
}

// Prazo dá à requisição o prazo que o cliente pediu no cabeçalho
// Enxame-Timeout — o grpc-timeout feito à mão —, até um teto, ou um
// padrão. O prazo segue no contexto até o banco.
func Prazo(padrao, teto time.Duration) Middleware {
	return func(prox http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter,
			r *http.Request) {
			d := padrao
			if v, err := time.ParseDuration(
				r.Header.Get("Enxame-Timeout")); err == nil && v > 0 {
				d = min(v, teto)
			}
			ctx, cancel := context.WithTimeout(r.Context(), d)
			defer cancel()
			prox.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// Autenticar resolve o token no namespace do cliente; sem token
// válido, 401.
func (a *API) Autenticar(prox http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter,
		r *http.Request) {
		recebido, _ := strings.CutPrefix(
			r.Header.Get("Authorization"), "Bearer ")
		for token, ns := range a.Tokens {
			if subtle.ConstantTimeCompare([]byte(recebido),
				[]byte(token)) == 1 {
				ctx := context.WithValue(r.Context(),
					chaveNamespace{}, ns)
				prox.ServeHTTP(w, r.WithContext(ctx))
				return
			}
		}
		escrever(w, http.StatusUnauthorized,
			Erro{"unauthenticated", "token ausente ou inválido"})
	})
}

// LimitarCorpo recusa corpos maiores que n bytes.
func LimitarCorpo(n int64) Middleware {
	return func(prox http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter,
			r *http.Request) {
			r.Body = http.MaxBytesReader(w, r.Body, n)
			prox.ServeHTTP(w, r)
		})
	}
}

// livro:fim middleware

// Registrar anota cada requisição: rota, status e duração.
func (a *API) Registrar(prox http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter,
		r *http.Request) {
		inicio := time.Now()
		rw := &comStatus{ResponseWriter: w, status: http.StatusOK}
		// O padrão da rota só é conhecido depois que o ServeMux escolhe
		// o handler, lá dentro; anotarRota o devolve por aqui.
		rota := new(string)
		ctx := context.WithValue(r.Context(), chaveRota{}, rota)
		prox.ServeHTTP(rw, r.WithContext(ctx))
		a.Log.InfoContext(r.Context(), "http",
			slog.String("metodo", r.Method),
			slog.String("rota", *rota),
			slog.Int("status", rw.status),
			slog.Duration("duracao", time.Since(inicio)))
	})
}

type comStatus struct {
	http.ResponseWriter
	status int
}

func (c *comStatus) WriteHeader(s int) {
	c.status = s
	c.ResponseWriter.WriteHeader(s)
}

type chaveRota struct{}

// anotarRota guarda o padrão da rota escolhida para o Registrar.
func anotarRota(prox http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter,
		r *http.Request) {
		if rota, ok := r.Context().Value(chaveRota{}).(*string); ok {
			*rota = r.Pattern
		}
		prox.ServeHTTP(w, r)
	})
}
