package scheduler_test

import (
	"slices"
	"testing"
	"time"

	"github.com/go-sob-pressao/enxame/internal/simulation/scheduler"
)

func execucao(seed uint64) []int {
	s := scheduler.New(seed)
	var ordem []int
	for i := range 50 {
		atraso := time.Duration(s.Rand().IntN(100)) * time.Millisecond
		s.After(atraso, func() { ordem = append(ordem, i) })
	}
	s.Run(time.Second)
	return ordem
}

func TestMesmaSeedMesmaExecucao(t *testing.T) {
	if !slices.Equal(execucao(8371), execucao(8371)) {
		t.Fatal("a mesma seed produziu execuções diferentes")
	}
	if slices.Equal(execucao(8371), execucao(8372)) {
		t.Fatal("seeds diferentes produziram a mesma execução")
	}
}
