//go:build defeito

package ingenuo

import (
	"testing"

	"github.com/go-sob-pressao/enxame/internal/store/storetest"
)

// O fake passa por todos os testes do serviço que o usava. Posto diante
// da suíte de contrato, mostra tudo o que prometia e não cumpria.
func TestFakeContraOContrato(t *testing.T) {
	storetest.Run(t, func(*testing.T) storetest.Store { return &Fake{} })
}
