package serde

import (
	"bytes"
	"encoding/json"
	"fmt"
)

// livro:inicio canonical

// Canonical devolve a forma canônica de um JSON: chaves de objeto em
// ordem, sem espaços, e cada número exatamente como veio. O decoder usa
// UseNumber: sem ele, todo número vira float64, e um inteiro acima de
// 2^53 — um id de pedido de 19 dígitos — perde os últimos dígitos.
func Canonical(data []byte) ([]byte, error) {
	if len(bytes.TrimSpace(data)) == 0 {
		data = []byte("{}")
	}
	d := json.NewDecoder(bytes.NewReader(data))
	d.UseNumber()
	var v any
	if err := d.Decode(&v); err != nil {
		return nil, fmt.Errorf("json inválido: %w", err)
	}
	if d.More() {
		return nil, fmt.Errorf("json inválido: dados depois do valor")
	}
	return json.Marshal(v) // mapas saem com as chaves ordenadas
}

// Equal diz se dois JSON representam o mesmo valor, número a número.
func Equal(a, b []byte) (bool, error) {
	ca, err := Canonical(a)
	if err != nil {
		return false, err
	}
	cb, err := Canonical(b)
	if err != nil {
		return false, err
	}
	return bytes.Equal(ca, cb), nil
}

// livro:fim canonical
