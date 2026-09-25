//go:build integration

package integration_test

import (
	"testing"

	"github.com/go-sob-pressao/enxame/internal/store/postgres"
	"github.com/go-sob-pressao/enxame/internal/worker/workflow/workflowtest"
	"github.com/go-sob-pressao/enxame/test/testutil"
)

func TestReplayNoPostgres(t *testing.T) {
	workflowtest.Run(t, func(t *testing.T) workflowtest.Store {
		return postgres.New(testutil.Postgres(t))
	})
}
