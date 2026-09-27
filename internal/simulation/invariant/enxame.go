package invariant

import (
	"fmt"

	"github.com/go-sob-pressao/enxame/internal/simulation/banco"
)

// livro:inicio invariantes-enxame

// Enxame acompanha uma execução do cluster simulado e registra toda
// violação das propriedades que o M5 promete:
//
//   - nenhum job perdido: terminado o cenário e curadas as falhas, todo
//     job foi concluído;
//   - uma conclusão por job: só uma tentativa conclui cada job;
//   - posse monotônica: numa partição, nenhuma reserva é feita com um
//     token menor que o de uma reserva anterior;
//   - ordem por chave: os jobs de uma chave concluem na ordem em que
//     foram enfileirados.
type Enxame struct {
	concluidos map[int]bool
	token      map[int]int64 // maior token que reservou na partição
	proximo    map[string]int
	Violacoes  []string
}

// NewEnxame cria o verificador.
func NewEnxame() *Enxame {
	return &Enxame{concluidos: map[int]bool{}, token: map[int]int64{},
		proximo: map[string]int{}}
}

// Reservou registra uma tentativa começada por no, com o token dado.
func (v *Enxame) Reservou(j banco.Job, no string, token int64) {
	if token < v.token[j.Particao] {
		v.falha("partição %d: %s reservou o job %d com o token %d, "+
			"depois de uma reserva com o %d", j.Particao, no, j.ID,
			token, v.token[j.Particao])
	}
	v.token[j.Particao] = max(v.token[j.Particao], token)
}

// Concluiu registra a conclusão de um job.
func (v *Enxame) Concluiu(j banco.Job, no string) {
	if v.concluidos[j.ID] {
		v.falha("job %d concluído duas vezes (a segunda por %s)",
			j.ID, no)
	}
	v.concluidos[j.ID] = true
	if j.Seq != v.proximo[j.Chave] {
		v.falha("chave %s: concluiu o job %d (posição %d) quando "+
			"esperava a posição %d", j.Chave, j.ID, j.Seq,
			v.proximo[j.Chave])
	}
	v.proximo[j.Chave] = j.Seq + 1
}

// Adquiriu e Soltou não têm regra própria: a posse é conferida nas
// reservas.
func (v *Enxame) Adquiriu(int, string, int64) {}

// Soltou também não tem regra própria.
func (v *Enxame) Soltou(int, string, int64) {}

// Fim confere, ao fim do cenário, que nenhum job ficou para trás; quem
// explica cada perdido é quem chama.
func (v *Enxame) Fim(b *banco.Banco,
	explicar func(j *banco.Job) string) {
	for _, j := range b.Jobs {
		if !v.concluidos[j.ID] {
			v.falha("job %d perdido (partição %d, chave %s): %s",
				j.ID, j.Particao, j.Chave, explicar(j))
		}
	}
}

// livro:fim invariantes-enxame

func (v *Enxame) falha(f string, a ...any) {
	v.Violacoes = append(v.Violacoes, fmt.Sprintf(f, a...))
}
