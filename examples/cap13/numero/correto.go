//go:build !defeito

package numero

import "github.com/go-sob-pressao/enxame/internal/store/serde"

// Canonical usa a forma canônica do Enxame, que preserva os números.
func Canonical(data []byte) ([]byte, error) {
	return serde.Canonical(data)
}
