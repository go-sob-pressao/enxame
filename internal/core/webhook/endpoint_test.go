package webhook_test

import (
	"errors"
	"testing"

	"github.com/go-sob-pressao/enxame/internal/core/webhook"
)

func TestValidar(t *testing.T) {
	ok := webhook.Endpoint{URL: "https://cliente.exemplo/hooks",
		SecretRef: "env:SEGREDO"}
	if err := webhook.Validar(ok); err != nil {
		t.Fatal(err)
	}
	for nome, e := range map[string]webhook.Endpoint{
		"sem esquema":  {URL: "cliente.exemplo", SecretRef: "x"},
		"ftp":          {URL: "ftp://cliente.exemplo", SecretRef: "x"},
		"sem host":     {URL: "https:///hooks", SecretRef: "x"},
		"sem segredo":  {URL: "https://cliente.exemplo"},
		"evento vazio": {URL: "https://c.exemplo", SecretRef: "x", EventTypes: []string{" "}},
	} {
		if err := webhook.Validar(e); !errors.Is(err, webhook.ErrInvalido) {
			t.Errorf("%s: %v", nome, err)
		}
	}
}
