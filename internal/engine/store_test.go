package engine_test

import (
	"github.com/go-sob-pressao/enxame/internal/engine"
	"github.com/go-sob-pressao/enxame/internal/store/memory"
	"github.com/go-sob-pressao/enxame/internal/store/postgres"
	"github.com/go-sob-pressao/enxame/internal/store/sqlite"
	"github.com/go-sob-pressao/enxame/internal/store/storetest"
)

// O contrato da suíte cobre o que o motor precisa, e as três
// implementações satisfazem o motor. Se alguém mudar uma assinatura, o
// erro aparece aqui, em tempo de compilação.
var (
	_ engine.Store = storetest.Store(nil)
	_ engine.Store = (*memory.Store)(nil)
	_ engine.Store = (*sqlite.Store)(nil)
	_ engine.Store = (*postgres.Store)(nil)
)
