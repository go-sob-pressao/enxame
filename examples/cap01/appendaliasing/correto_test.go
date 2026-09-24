//go:build !defeito

package main

import (
	"slices"
	"testing"
)

func TestVariacoesIndependentes(t *testing.T) {
	a, b := derivar([]int{1, 2, 3})
	if !slices.Equal(a, []int{1, 2, 3, 100}) ||
		!slices.Equal(b, []int{1, 2, 3, 200}) {
		t.Fatalf("a=%v b=%v", a, b)
	}
}
