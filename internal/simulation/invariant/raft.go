package invariant

import (
	"fmt"
	"slices"

	"github.com/go-sob-pressao/enxame/internal/cluster/raft"
)

// livro:inicio invariantes-raft

// Raft acompanha uma execução do Raft e registra toda violação das
// propriedades de segurança do algoritmo:
//
//   - segurança de eleição: no máximo um líder por termo;
//   - segurança da máquina de estados: nenhum índice aplicado com dois
//     valores diferentes, em nó nenhum;
//   - log matching: se dois logs têm uma entrada com o mesmo índice e o
//     mesmo termo, eles são idênticos até ali.
type Raft struct {
	lideres   map[raft.Term]raft.NodeID
	porIndice map[raft.Index]raft.Entry
	Violacoes []string
}

// NewRaft cria o verificador.
func NewRaft() *Raft {
	return &Raft{
		lideres:   map[raft.Term]raft.NodeID{},
		porIndice: map[raft.Index]raft.Entry{},
	}
}

// Lider registra que id se viu líder no termo t.
func (v *Raft) Lider(t raft.Term, id raft.NodeID) {
	if outro, ok := v.lideres[t]; ok && outro != id {
		v.falha("dois líderes no termo %d: %d e %d", t, outro, id)
		return
	}
	v.lideres[t] = id
}

// Aplicada registra que o nó aplicou a entrada e.
func (v *Raft) Aplicada(id raft.NodeID, e raft.Entry) {
	if outra, ok := v.porIndice[e.Index]; ok &&
		(outra.Term != e.Term || !slices.Equal(outra.Data, e.Data)) {
		v.falha(
			"nó %d aplicou no índice %d %q (termo %d); "+
				"já aplicado %q (termo %d)",
			id,
			e.Index,
			e.Data,
			e.Term,
			outra.Data,
			outra.Term,
		)
		return
	}
	v.porIndice[e.Index] = e
}

// LogMatching confere a propriedade entre todos os pares de logs.
func (v *Raft) LogMatching(logs map[raft.NodeID][]raft.Entry) {
	ids := make([]raft.NodeID, 0, len(logs))
	for id := range logs {
		ids = append(ids, id)
	}
	slices.Sort(ids)
	for i, a := range ids {
		for _, b := range ids[i+1:] {
			v.compararLogs(a, logs[a], b, logs[b])
		}
	}
}

// livro:fim invariantes-raft

func (v *Raft) compararLogs(
	a raft.NodeID,
	la []raft.Entry,
	b raft.NodeID,
	lb []raft.Entry,
) {
	pb := map[raft.Index]raft.Entry{}
	for _, e := range lb {
		pb[e.Index] = e
	}
	// maior índice comum com o mesmo termo: daí para trás, tudo igual
	var ultimo raft.Index
	for _, e := range la {
		if o, ok := pb[e.Index]; ok && o.Term == e.Term {
			ultimo = e.Index
		}
	}
	for _, e := range la {
		if e.Index > ultimo {
			break
		}
		o, ok := pb[e.Index]
		if ok && (o.Term != e.Term || !slices.Equal(o.Data, e.Data)) {
			v.falha(
				"log matching: nós %d e %d concordam no índice %d, "+
					"divergem no %d",
				a,
				b,
				ultimo,
				e.Index,
			)
			return
		}
	}
}

func (v *Raft) falha(format string, args ...any) {
	v.Violacoes = append(v.Violacoes, fmt.Sprintf(format, args...))
}
