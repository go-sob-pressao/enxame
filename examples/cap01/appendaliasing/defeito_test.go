//go:build defeito

package main

import "testing"

func TestSegundoAppendSobrescreve(t *testing.T) {
	a, _ := derivar([]int{1, 2, 3})
	if a[3] != 200 {
		t.Fatalf("o enigma não se reproduziu: a[3]=%d", a[3])
	}
	t.Log("a[3] vale 200: o append de b escreveu no array de a")
}
