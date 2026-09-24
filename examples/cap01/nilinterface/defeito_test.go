//go:build defeito

package main

import "testing"

func TestNomeValidoDevolveErroNaoNulo(t *testing.T) {
	err := validar("Ana")
	if err == nil {
		t.Fatal("o enigma não se reproduziu: err deveria ser não nulo")
	}
	t.Logf("err != nil, e imprime como %v", err)
}
