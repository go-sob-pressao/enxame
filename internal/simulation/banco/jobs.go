package banco

import (
	"slices"
	"time"

	"github.com/go-sob-pressao/enxame/internal/core/id"
)

// Enfileirar cria um job com chave de ordem; a partição vem da chave,
// pela função de produção.
func (b *Banco) Enfileirar(chave string, seq int) *Job {
	j := &Job{ID: len(b.Jobs) + 1, Particao: id.Particao(chave),
		Chave: chave, Seq: seq}
	b.Jobs = append(b.Jobs, j)
	return j
}

// cabeca é a condição da reserva por chave: nenhum job mais antigo da
// mesma chave por terminar.
func (b *Banco) cabeca(j *Job) bool {
	for _, a := range b.Jobs[:j.ID-1] {
		if a.Chave == j.Chave && a.Estado != Concluido {
			return false
		}
	}
	return true
}

// livro:inicio reserva-modelo

// Reservar é a transação de claimDoDono: acha o primeiro job esperando,
// cabeça da chave, nas partições dadas; confere, na mesma transação, o
// token da partição dele; e começa uma tentativa. Devolve o job, ou a
// partição cuja cerca recusou o token.
func (b *Banco) Reservar(no string,
	tokens map[int]int64) (j Job, achou bool, perdida int) {
	for _, c := range b.Jobs {
		token, meu := tokens[c.Particao]
		if !meu || c.Estado != Esperando || !b.cabeca(c) {
			continue
		}
		if b.posses[c.Particao].rangeID != token {
			return Job{}, false, c.Particao // ErrCercado
		}
		c.Estado, c.Batida = Rodando, b.Agora()
		c.Tentativa++
		b.avisar(func(o Observador) { o.Reservou(*c, no, token) })
		return *c, true, -1
	}
	return Job{}, false, -1
}

// livro:fim reserva-modelo

// BaterJob é a batida da tentativa em curso.
func (b *Banco) BaterJob(jid, tentativa int) {
	j := b.Jobs[jid-1]
	if j.Estado == Rodando && j.Tentativa == tentativa {
		j.Batida = b.Agora()
	}
}

// Concluir é Complete com a cerca da tentativa: só a tentativa vigente
// conclui o job.
func (b *Banco) Concluir(no string, jid, tentativa int) bool {
	j := b.Jobs[jid-1]
	if j.Estado != Rodando || j.Tentativa != tentativa {
		return false
	}
	j.Estado = Concluido
	b.avisar(func(o Observador) { o.Concluiu(*j, no) })
	return true
}

// Resgatar devolve à fila as tentativas sem batida desde antes de
// limite, só nas partições cujo token confere — o Rescue do dono.
func (b *Banco) Resgatar(tokens map[int]int64, limite time.Time) int {
	n := 0
	for _, j := range b.Jobs {
		token, meu := tokens[j.Particao]
		if meu && j.Estado == Rodando && j.Batida.Before(limite) &&
			b.posses[j.Particao].rangeID == token {
			j.Estado = Esperando
			n++
		}
	}
	return n
}

// EmCurso conta as tentativas em execução em cada partição de ps.
func (b *Banco) EmCurso(ps []int) map[int]int {
	n := map[int]int{}
	for _, j := range b.Jobs {
		if j.Estado == Rodando && slices.Contains(ps, j.Particao) {
			n[j.Particao]++
		}
	}
	return n
}
