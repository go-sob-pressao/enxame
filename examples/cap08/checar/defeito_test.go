//go:build defeito

package checar

import "testing"

// Rode com -race: o detector fica em silêncio — não há corrida de
// DADOS, cada acesso ao sync.Map é sincronizado. A corrida é de LÓGICA.
func TestMaisDeUmaConexao(t *testing.T) {
	pior := int64(1)
	for range 100 {
		pior = max(pior, obterEmParalelo())
	}
	if pior == 1 {
		t.Skip(
			"nenhuma execução intercalou Load e Store; aumente as repetições",
		)
	}
	t.Logf("até %d conexões abertas para o mesmo endpoint", pior)
}
