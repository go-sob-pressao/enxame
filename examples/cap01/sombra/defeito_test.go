//go:build defeito

package main

import "testing"

func TestFalhaDesaparece(t *testing.T) {
	total, err := somarEstoque([]int{1, 2, 3, 4})
	if err != nil {
		t.Fatalf("o enigma não se reproduziu: err=%v", err)
	}
	t.Logf(
		"total=%d e err=nil, embora a consulta do item 3 tenha falhado",
		total,
	)
}
