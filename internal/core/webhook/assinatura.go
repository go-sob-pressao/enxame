package webhook

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// ErrSegredo indica um segredo fora do formato whsec_<base64>.
var ErrSegredo = errors.New("segredo de webhook inválido")

// livro:inicio assinatura

// Assinar calcula a assinatura de uma entrega no formato Standard
// Webhooks: HMAC-SHA256 de "id.timestamp.corpo", com a chave que o
// segredo whsec_<base64> carrega, em base64, precedida da versão "v1,".
// O id é o da mensagem — o mesmo em todas as tentativas, para que quem
// recebe deduplique —, e o timestamp, o do envio, para que quem recebe
// recuse uma entrega antiga reenviada por outro.
func Assinar(
	chave []byte,
	msgID string,
	enviadoEm time.Time,
	corpo []byte,
) string {
	h := hmac.New(sha256.New, chave)
	h.Write([]byte(msgID + "." +
		strconv.FormatInt(enviadoEm.Unix(), 10) + "."))
	h.Write(corpo)
	return "v1," + base64.StdEncoding.EncodeToString(h.Sum(nil))
}

// livro:fim assinatura

// Chave extrai a chave do segredo whsec_<base64>. O padrão pede de 24
// a 64 bytes aleatórios.
func Chave(segredo string) ([]byte, error) {
	b64, ok := strings.CutPrefix(segredo, "whsec_")
	if !ok {
		return nil, fmt.Errorf("%w: falta o prefixo whsec_", ErrSegredo)
	}
	chave, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrSegredo, err)
	}
	if len(chave) < 24 || len(chave) > 64 {
		return nil, fmt.Errorf("%w: %d bytes; o padrão pede 24 a 64",
			ErrSegredo, len(chave))
	}
	return chave, nil
}

// Resultado é o que o status da resposta diz sobre a entrega.
type Resultado int

// Os resultados possíveis de uma tentativa.
const (
	Entregue   Resultado = iota // 2xx
	Falhou                      // repetir, com backoff
	Desacelere                  // 429, 502, 504: repetir, mais devagar
	Desative                    // 410: o endpoint pediu para sair
)

// Classificar segue a especificação: 2xx é sucesso; 410 desativa o
// endpoint; 429, 502 e 504 pedem calma; o resto — 3xx incluído — é
// falha.
func Classificar(status int) Resultado {
	switch {
	case status >= 200 && status <= 299:
		return Entregue
	case status == 410:
		return Desative
	case status == 429 || status == 502 || status == 504:
		return Desacelere
	}
	return Falhou
}
