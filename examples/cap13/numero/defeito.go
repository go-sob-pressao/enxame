//go:build defeito

package numero

import "encoding/json"

// livro:inicio canonical-defeito

// Canonical decodifica para any e codifica de novo: chaves ordenadas,
// sem espaços. Todo número vira float64 no caminho.
func Canonical(data []byte) ([]byte, error) {
	var v any
	if err := json.Unmarshal(data, &v); err != nil {
		return nil, err
	}
	return json.Marshal(v)
}

// livro:fim canonical-defeito
