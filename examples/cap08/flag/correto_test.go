//go:build !defeito

package flag

import "testing"

func TestAtomicPara(t *testing.T) {
	if !pararEmUmSegundo() {
		t.Fatal("não parou")
	}
}
