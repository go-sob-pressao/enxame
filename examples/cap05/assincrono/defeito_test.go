//go:build defeito

package main

import (
	"context"
	"errors"
	"testing"
)

func TestProcessamentoHerdaCancelamento(t *testing.T) {
	if err := resultado(t); !errors.Is(err, context.Canceled) {
		t.Fatalf("o enigma não se reproduziu: err=%v", err)
	}
}
