package versao_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/go-sob-pressao/enxame/pkg/workflow"
)

// ReciboV1 é o resultado do passo "cobrar", como o código de ontem o
// gravou; ReciboV2 é o de hoje, com o campo renomeado.
type ReciboV1 struct {
	ID string `json:"id"`
}

type ReciboV2 struct {
	ID string `json:"recibo_id"`
}

// O run de ontem gravou {"id":"R-1"}. O código de hoje relê o passo
// com o tipo novo: nenhum erro, e o recibo volta vazio.
func TestCampoRenomeadoVoltaVazio(t *testing.T) {
	c := novo()
	id := c.iniciar(t)
	ontem := func(wc *workflow.Context, _ json.RawMessage) (any, error) {
		_, err := workflow.Step(wc, "cobrar",
			func(context.Context) (ReciboV1, error) {
				return ReciboV1{ID: "R-1"}, nil
			})
		if err != nil {
			return nil, err
		}
		return nil, workflow.Sleep(wc, "esperar", 48*time.Hour)
	}
	var relido ReciboV2
	hoje := func(wc *workflow.Context, _ json.RawMessage) (any, error) {
		r, err := workflow.Step(wc, "cobrar",
			func(context.Context) (ReciboV2, error) {
				t.Fatal("o passo gravado não deveria rodar")
				return ReciboV2{}, nil
			})
		relido = r
		if err != nil {
			return nil, err
		}
		return nil, workflow.Sleep(wc, "esperar", 48*time.Hour)
	}
	if _, err := c.avancar(t, ontem, id); err != nil {
		t.Fatal(err)
	}
	c.agora = c.agora.Add(time.Hour) // deploy; o run é reexecutado
	if _, err := c.avancar(t, hoje, id); err != nil {
		t.Fatal(err)
	}
	t.Logf("recibo relido: %+v", relido)
	if relido.ID != "" {
		t.Fatalf("esperava o campo vazio: %+v", relido)
	}
}
