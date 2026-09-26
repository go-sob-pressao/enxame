package coordinator_test

import (
	"testing"

	"github.com/go-sob-pressao/enxame/internal/cluster/coordinator"
)

func contagem(a coordinator.Assignment) map[coordinator.NodeID]int {
	c := map[coordinator.NodeID]int{}
	for _, o := range a.Owners {
		c[o]++
	}
	return c
}

func TestDistribuicaoBalanceada(t *testing.T) {
	a := coordinator.Distribute(
		[]coordinator.NodeID{"a", "b", "c"},
		coordinator.Assignment{},
	)
	for n, q := range contagem(a) {
		if q < 170 || q > 171 {
			t.Fatalf("nó %s com %d partições", n, q)
		}
	}
}

// Um nó sai: só as partições dele mudam de dono.
func TestSaidaMexeSoNoQueEraDeQuemSaiu(t *testing.T) {
	antes := coordinator.Distribute(
		[]coordinator.NodeID{"a", "b", "c", "d"},
		coordinator.Assignment{},
	)
	depois := coordinator.Distribute(
		[]coordinator.NodeID{"a", "b", "c"},
		antes,
	)
	if m := coordinator.Moved(antes, depois); m != 128 {
		t.Fatalf(
			"%d partições mudaram de dono; esperado 128 (as de d)",
			m,
		)
	}
	if depois.Epoch != antes.Epoch+1 {
		t.Fatalf("época %d → %d", antes.Epoch, depois.Epoch)
	}
}

// Um nó entra: ele recebe a cota dele, tirada dos que passaram da
// deles.
func TestEntradaMexeSoNaCotaDoNovo(t *testing.T) {
	antes := coordinator.Distribute(
		[]coordinator.NodeID{"a", "b"},
		coordinator.Assignment{},
	)
	depois := coordinator.Distribute(
		[]coordinator.NodeID{"a", "b", "c", "d"},
		antes,
	)
	if m := coordinator.Moved(antes, depois); m != 256 {
		t.Fatalf(
			"%d partições mudaram; esperado 256 (as cotas de c e d)",
			m,
		)
	}
}
