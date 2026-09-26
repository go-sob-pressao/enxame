package http

import (
	"context"
	"errors"
	"net"
	"net/http"
	"time"
)

func (a *API) vivo(w http.ResponseWriter, _ *http.Request) {
	escrever(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (a *API) pronto(w http.ResponseWriter, _ *http.Request) {
	if a.parando.Load() {
		escrever(w, http.StatusServiceUnavailable,
			map[string]string{"status": "encerrando"})
		return
	}
	escrever(w, http.StatusOK, map[string]string{"status": "ok"})
}

// Desligamento configura o encerramento gracioso.
type Desligamento struct {
	// Aviso é quanto tempo o /readyz responde 503 antes de o servidor
	// parar de aceitar conexões: o tempo de o balanceador tirar o nó
	// da rotação.
	Aviso time.Duration
	// Prazo é o teto para as requisições em andamento terminarem.
	Prazo time.Duration
}

// livro:inicio desligamento

// Servir atende em lis até ctx terminar, e então desliga em ordem:
// primeiro avisa (o /readyz passa a responder 503 e o balanceador tira
// o nó da rotação), depois acorda os long-polls, depois espera as
// requisições em andamento — até um prazo. O prazo precisa caber no
// tempo que o orquestrador dá entre o SIGTERM e o SIGKILL.
func (a *API) Servir(
	ctx context.Context,
	lis net.Listener,
	d Desligamento,
) error {
	srv := &http.Server{Handler: a.Handler(),
		ReadHeaderTimeout: 5 * time.Second,
		BaseContext: func(net.Listener) context.Context {
			return context.WithoutCancel(ctx)
		}}
	srv.RegisterOnShutdown(func() { close(a.encerrando) })
	erro := make(chan error, 1)
	go func() { erro <- srv.Serve(lis) }()
	select {
	case err := <-erro:
		return err
	case <-ctx.Done():
	}
	a.parando.Store(true)
	<-time.After(d.Aviso)
	base := context.WithoutCancel(ctx)
	fim, cancel := context.WithTimeout(base, d.Prazo)
	defer cancel()
	err := srv.Shutdown(fim) // para de aceitar; espera as ativas
	if e := <-erro; !errors.Is(e, http.ErrServerClosed) {
		return errors.Join(err, e)
	}
	return err
}

// livro:fim desligamento
