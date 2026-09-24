//go:build defeito

package flag

import "testing"

// O memory model não garante que o laço observe a escrita: sem
// happens-before, "nunca parar" é um resultado permitido. O compilador
// gc do Go 1.27 em amd64 relê o campo a cada volta, e na prática o laço
// para — hoje, nesta arquitetura. Rode com -race: o detector acusa a
// corrida, que é o que importa.
func TestBoolSemGarantia(t *testing.T) {
	if pararEmUmSegundo() {
		t.Log(
			"parou nesta execução — sorte da implementação, não garantia da linguagem",
		)
		return
	}
	t.Log("1 s depois de Parar(), o laço continua girando")
}
