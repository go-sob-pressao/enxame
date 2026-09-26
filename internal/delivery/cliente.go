package delivery

import (
	"io"
	"net"
	"net/http"
	"time"
)

// livro:inicio cliente

// NovoCliente devolve o cliente HTTP da entrega, com prazo em cada
// fase. O http.Client zero não tem prazo nenhum: um endpoint que aceita
// a conexão e não responde seguraria o worker para sempre.
func NovoCliente() *http.Client {
	return &http.Client{
		// O teto da tentativa inteira: conexão, envio, resposta,
		// corpo.
		// A especificação do Standard Webhooks sugere de 15 a 30 s.
		Timeout: 15 * time.Second,
		Transport: &http.Transport{
			DialContext: (&net.Dialer{
				Timeout: 5 * time.Second, // conexão TCP
			}).DialContext,
			TLSHandshakeTimeout:   5 * time.Second,
			ResponseHeaderTimeout: 10 * time.Second, // o primeiro byte
			IdleConnTimeout:       90 * time.Second,
			MaxIdleConnsPerHost:   16, // reuso de conexão por endpoint
		},
		// Redirecionamento conta como falha: o endpoint é o cadastrado.
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
}

// lerResposta guarda o começo do corpo, para diagnóstico, e descarta o
// resto até um teto. Ler até o fim é o que devolve a conexão ao pool
// para ser reusada; o teto impede que um endpoint mande um gigabyte.
// Quem chama fecha o corpo.
func lerResposta(r io.Reader) string {
	trecho := make([]byte, 512)
	n, _ := io.ReadFull(r, trecho)
	_, _ = io.CopyN(io.Discard, r, 64<<10)
	return string(trecho[:n])
}

// livro:fim cliente
