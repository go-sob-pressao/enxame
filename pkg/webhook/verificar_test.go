package webhook_test

import (
	"encoding/base64"
	"errors"
	"net/http"
	"strconv"
	"testing"
	"time"

	standardwebhooks "github.com/standard-webhooks/standard-webhooks/libraries/go"

	nucleo "github.com/go-sob-pressao/enxame/internal/core/webhook"
	"github.com/go-sob-pressao/enxame/pkg/webhook"
)

var segredo = "whsec_" + base64.StdEncoding.EncodeToString(
	[]byte("uma chave de trinta e dois bytes"))

func cabecalhos(id string, t time.Time, assinatura string) http.Header {
	h := http.Header{}
	h.Set("webhook-id", id)
	h.Set("webhook-timestamp", strconv.FormatInt(t.Unix(), 10))
	h.Set("webhook-signature", assinatura)
	return h
}

// A assinatura do Enxame é verificada pela biblioteca de referência do
// Standard Webhooks, e a da biblioteca, pelo Enxame.
func TestCompativelComAReferencia(t *testing.T) {
	agora := time.Now()
	corpo := []byte(`{"type":"pedido.pago","data":{"id":42}}`)
	chave, err := nucleo.Chave(segredo)
	if err != nil {
		t.Fatal(err)
	}
	ref, err := standardwebhooks.NewWebhook(segredo)
	if err != nil {
		t.Fatal(err)
	}
	nossa := nucleo.Assinar(chave, "msg_1", agora, corpo)
	if err := ref.Verify(corpo,
		cabecalhos("msg_1", agora, nossa)); err != nil {
		t.Fatalf("a referência recusou a nossa: %v", err)
	}
	dela, err := ref.Sign("msg_2", agora, corpo)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := webhook.Verificar(segredo,
		cabecalhos("msg_2", agora, dela), corpo, agora); err != nil {
		t.Fatalf("recusamos a da referência: %v", err)
	}
}

func TestVerificarRecusa(t *testing.T) {
	agora := time.Now()
	corpo := []byte(`{"id":42}`)
	chave, _ := nucleo.Chave(segredo)
	boa := nucleo.Assinar(chave, "msg_1", agora, corpo)
	casos := []struct {
		nome  string
		h     http.Header
		corpo []byte
		erro  error
	}{
		{"corpo alterado", cabecalhos("msg_1", agora, boa),
			[]byte(`{"id":43}`), webhook.ErrAssinatura},
		{"outro id", cabecalhos("msg_9", agora, boa), corpo,
			webhook.ErrAssinatura},
		{"antiga", cabecalhos("msg_1", agora.Add(-10*time.Minute),
			nucleo.Assinar(chave, "msg_1",
				agora.Add(-10*time.Minute), corpo)), corpo,
			webhook.ErrAntigo},
	}
	for _, c := range casos {
		if _, err := webhook.Verificar(segredo, c.h, c.corpo,
			agora); !errors.Is(err, c.erro) {
			t.Errorf("%s: %v", c.nome, err)
		}
	}
	// Duas assinaturas, durante a troca de segredo: vale a que casar.
	h := cabecalhos("msg_1", agora, "v1,velha "+boa)
	if id, err := webhook.Verificar(segredo, h, corpo, agora); err != nil ||
		id != "msg_1" {
		t.Fatalf("rotação: %q %v", id, err)
	}
}
