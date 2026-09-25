//go:build defeito

package auditoria_test

import (
	"testing"

	"github.com/go-sob-pressao/enxame/examples/cap14/auditoria"
)

// Reconstruir o estado a partir da auditoria dá um resultado; o estado
// gravado diz outro. Qual dos dois é o verdadeiro? Nada no sistema sabe
// responder — e nada no sistema percebeu que eles divergiram.
func TestAuditoriaContaOutraHistoria(t *testing.T) {
	j := &auditoria.Job{Estado: "available"}
	j.Iniciar()
	j.Resgatar()
	reconstruido := "available"
	for _, linha := range j.Auditoria {
		if linha == "tentativa iniciada" {
			reconstruido = "running"
		}
	}
	if reconstruido != j.Estado {
		t.Fatalf("auditoria diz %s; estado diz %s",
			reconstruido, j.Estado)
	}
}
