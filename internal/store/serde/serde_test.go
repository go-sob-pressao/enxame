package serde_test

import (
	"testing"

	"github.com/go-sob-pressao/enxame/internal/store/serde"
)

func TestEqual(t *testing.T) {
	casos := []struct {
		a, b string
		quer bool
	}{
		{`{"a":1,"b":[1,2]}`, `{ "b": [1, 2], "a": 1 }`, true},
		{`{"pedido":9007199254740993}`, `{"pedido":9007199254740992}`, false},
		{`{"n":1.0}`, `{"n":1}`, false}, // literais diferentes
		{``, `{}`, true},
	}
	for _, c := range casos {
		got, err := serde.Equal([]byte(c.a), []byte(c.b))
		if err != nil || got != c.quer {
			t.Errorf("Equal(%s, %s) = %v, %v", c.a, c.b, got, err)
		}
	}
}
