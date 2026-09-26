package workflow

import (
	"encoding/json"
	"fmt"
)

// KindVersion é o marcador gravado por Version.
const KindVersion Kind = "version"

// DefaultVersion é a versão que Version devolve aos runs gravados por
// código anterior à mudança.
const DefaultVersion = 0

// livro:inicio version

// Version marca, no histórico, uma mudança no código do workflow. Um
// run novo grava maximo e segue o caminho novo. Um run antigo — cuja
// próxima posição já guarda outro passo, gravado pelo código de antes
// — recebe DefaultVersion sem ocupar posição nenhuma, e segue o caminho
// antigo. minimo acima de DefaultVersion declara que o código não sabe
// mais continuar runs antigos: eles param com erro, em vez de seguir um
// caminho que não existe mais.
func Version(c *Context, mudanca string, minimo, maximo int) (int, error) {
	r, gravado := c.hist.Lookup(c.seq + 1)
	switch {
	case gravado && r.Kind == KindVersion && r.Name == mudanca:
		c.seq++
		var v int
		if err := json.Unmarshal(r.Output, &v); err != nil {
			return 0, err
		}
		return v, suportada(c.seq, mudanca, v, minimo, maximo)
	case gravado: // o run é anterior à mudança
		return DefaultVersion,
			suportada(c.seq+1, mudanca, DefaultVersion, minimo, maximo)
	}
	r, _, err := c.proximo(mudanca, KindVersion)
	if err != nil {
		return 0, err
	}
	if r.Output, err = json.Marshal(maximo); err != nil {
		return 0, err
	}
	return maximo, c.hist.Append(c.ctx, r)
}

// livro:fim version

func suportada(seq int, mudanca string, v, minimo, maximo int) error {
	if v < minimo || v > maximo {
		return &NonDeterministicError{Seq: seq, Recorded: fmt.Sprintf(
			"%s versão %d", mudanca, v), RecordedKind: KindVersion,
			Requested: fmt.Sprintf("%s versões %d a %d", mudanca,
				minimo, maximo), RequestedKind: KindVersion}
	}
	return nil
}
