//go:build integration

package integration_test

import (
	"testing"

	"github.com/go-sob-pressao/enxame/internal/store/postgres"
	"github.com/go-sob-pressao/enxame/internal/store/storetest"
	"github.com/go-sob-pressao/enxame/test/testutil"
)

// livro:inicio contrato-postgres

func TestContratoPostgres(t *testing.T) {
	storetest.Run(t, func(t *testing.T) storetest.Store {
		return postgres.New(testutil.Postgres(t))
	})
}

// livro:fim contrato-postgres
