package memory_test

import (
	"testing"

	"github.com/go-sob-pressao/enxame/internal/store/memory"
	"github.com/go-sob-pressao/enxame/internal/store/storetest"
)

// livro:inicio contrato-memoria

func TestContrato(t *testing.T) {
	storetest.Run(t, func(*testing.T) storetest.Store {
		return memory.New()
	})
}

// livro:fim contrato-memoria
