package workflow_test

import (
	"testing"

	"github.com/go-sob-pressao/enxame/internal/store/memory"
	"github.com/go-sob-pressao/enxame/internal/worker/workflow/workflowtest"
)

func TestReplayEmMemoria(t *testing.T) {
	workflowtest.Run(t, func(*testing.T) workflowtest.Store {
		return memory.NewWorkflows()
	})
}
