package webhook

import (
	"crypto/hmac"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	nucleo "github.com/go-sob-pressao/enxame/internal/core/webhook"
)

// Erros da verificação.
var (
	ErrAssinatura = errors.New("assinatura inválida")
	ErrAntigo     = errors.New("timestamp fora da tolerância")
)

// Tolerancia é quanto o timestamp de uma entrega pode divergir do
// relógio de quem recebe.
const Tolerancia = 5 * time.Minute

// livro:inicio verificar

// Verificar confere uma entrega recebida do Enxame: o timestamp dentro
// da tolerância — para recusar uma entrega antiga, reenviada por quem
// a capturou — e pelo menos uma das assinaturas do cabeçalho, que pode
// trazer várias durante a troca de segredo. Devolve o webhook-id, a
// chave para deduplicar.
func Verificar(
	segredo string,
	h http.Header,
	corpo []byte,
	agora time.Time,
) (string, error) {
	chave, err := nucleo.Chave(segredo)
	if err != nil {
		return "", err
	}
	id := h.Get("webhook-id")
	seg, err := strconv.ParseInt(h.Get("webhook-timestamp"), 10, 64)
	if err != nil || id == "" {
		return "", ErrAssinatura
	}
	enviado := time.Unix(seg, 0)
	if agora.Sub(enviado).Abs() > Tolerancia {
		return "", ErrAntigo
	}
	esperada := nucleo.Assinar(chave, id, enviado, corpo)
	for recebida := range strings.FieldsSeq(h.Get("webhook-signature")) {
		if hmac.Equal([]byte(recebida), []byte(esperada)) {
			return id, nil
		}
	}
	return "", ErrAssinatura
}

// livro:fim verificar
