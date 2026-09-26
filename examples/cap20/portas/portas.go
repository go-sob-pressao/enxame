// Package portas é o enigma do Capítulo 20: o webhook que esgotava as
// portas efêmeras porque ninguém lia o corpo da resposta.
package portas

import (
	"context"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync/atomic"
	"time"
)

// Endpoint é o receptor do cliente: responde 200 com uma página de
// "obrigado" de 16 KiB, e conta as conexões TCP novas que recebe.
type Endpoint struct {
	*httptest.Server
	Conexoes atomic.Int64
}

// NovoEndpoint sobe o receptor. Lento, ele manda o começo da página,
// espera 100 ms e manda o resto.
func NovoEndpoint(lento bool) *Endpoint {
	e := &Endpoint{}
	pagina := strings.Repeat("obrigado! ", 16<<10/10)
	e.Server = httptest.NewUnstartedServer(http.HandlerFunc(
		func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Length", strconv.Itoa(len(pagina)))
			resto := pagina
			if lento {
				_, _ = io.WriteString(w, resto[:1024])
				w.(http.Flusher).Flush()
				<-time.After(100 * time.Millisecond)
				resto = resto[1024:]
			}
			_, _ = io.WriteString(w, resto)
		}))
	e.Config.ConnState = func(_ net.Conn, s http.ConnState) {
		if s == http.StateNew {
			e.Conexoes.Add(1)
		}
	}
	e.Start()
	return e
}

// livro:inicio portas

// EntregarIngenuo só quer o status: fecha o corpo sem ler.
func EntregarIngenuo(c *http.Client, url string) (int, error) {
	resp, err := postar(c, url)
	if err != nil {
		return 0, err
	}
	resp.Body.Close()
	return resp.StatusCode, nil
}

// Entregar lê o corpo até o fim — com teto — antes de fechar: é o que
// devolve a conexão ao pool do Transport para a próxima entrega.
func Entregar(c *http.Client, url string) (int, error) {
	resp, err := postar(c, url)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	_, _ = io.CopyN(io.Discard, resp.Body, 64<<10)
	return resp.StatusCode, nil
}

// livro:fim portas

// EntregarSemFechar esquece o corpo de vez: nem lê, nem fecha.
func EntregarSemFechar(c *http.Client, url string) (int, error) {
	resp, err := postar(c, url) //nolint:bodyclose // o defeito
	if err != nil {
		return 0, err
	}
	return resp.StatusCode, nil
}

func postar(c *http.Client, url string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(context.Background(),
		http.MethodPost, url,
		strings.NewReader(`{"type":"pedido.pago"}`))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	return c.Do(req)
}
