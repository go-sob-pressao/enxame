package dispatcher_test

import (
	"testing"

	"go.uber.org/goleak"
)

// livro:inicio goleak-testmain

// TestMain verifica, depois de TODOS os testes do pacote, que nenhuma
// goroutine iniciada por eles continua viva.
func TestMain(m *testing.M) {
	goleak.VerifyTestMain(m)
}

// livro:fim goleak-testmain
