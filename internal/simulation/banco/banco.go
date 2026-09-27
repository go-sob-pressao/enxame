// Package banco é o modelo, em memória, das tabelas que o cluster do
// Enxame usa no PostgreSQL: partition_lease, cluster_member,
// coord_leader, coord_assignment e job. Cada método é uma instrução
// (ou uma transação) do código de produção, com a mesma semântica e o
// mesmo relógio — o do banco. Não há concorrência aqui dentro: o
// escalonador da simulação executa um evento de cada vez, e cada
// método é atômico por construção.
//
// Camada: internal/simulation
// Introduzido no livro: Cap. 27 — ver docs/mapa-capitulos.md
package banco

import (
	"maps"
	"slices"
	"time"

	"github.com/go-sob-pressao/enxame/internal/cluster/coordinator"
)

// Estado de um job no modelo.
type Estado int

// Os estados que importam para a simulação.
const (
	Esperando Estado = iota
	Rodando
	Concluido
)

// Job é uma linha da tabela job, com o que a simulação usa dela.
type Job struct {
	ID        int
	Particao  int
	Chave     string
	Seq       int // posição do job na fila da chave
	Estado    Estado
	Tentativa int
	Batida    time.Time // última batida da tentativa em curso
}

type posse struct {
	dono    string
	rangeID int64
	expira  time.Time
}

type membro struct {
	inicio, batida time.Time
}

// Banco é o modelo. Agora é o relógio do banco: todo instante gravado
// sai dele, nunca do relógio de um nó.
type Banco struct {
	Agora func() time.Time

	posses  [coordinator.NumPartitions]posse
	membros map[coordinator.NodeID]*membro
	lider   struct {
		no     coordinator.NodeID
		termo  int64
		expira time.Time
	}
	mapa coordinator.Assignment
	Jobs []*Job // em ordem de id: o id cresce com a criação

	// Observador, se houver, é avisado de cada escrita que as
	// invariantes precisam ver.
	Observador Observador
}

// Observador recebe as escritas que as invariantes conferem.
type Observador interface {
	Reservou(j Job, no string, token int64)
	Concluiu(j Job, no string)
	Adquiriu(p int, no string, token int64)
	Soltou(p int, no string, token int64)
}

// Novo cria o banco vazio.
func Novo(agora func() time.Time) *Banco {
	return &Banco{Agora: agora,
		membros: map[coordinator.NodeID]*membro{}}
}

// livro:inicio lease-modelo

// AdquirirVarias é o UPDATE de partition.Lease.AdquirirVarias: toma
// cada partição livre, vencida ou já deste nó, e incrementa o
// rangeIDid.
func (b *Banco) AdquirirVarias(no string, dur time.Duration,
	ps []int) map[int]int64 {
	agora := b.Agora()
	ganhas := map[int]int64{}
	for _, p := range slices.Sorted(slices.Values(ps)) {
		l := &b.posses[p]
		if l.dono == "" || l.dono == no || l.expira.Before(agora) {
			l.dono, l.expira = no, agora.Add(dur)
			l.rangeID++
			ganhas[p] = l.rangeID
			t := l.rangeID
			b.avisar(func(o Observador) { o.Adquiriu(p, no, t) })
		}
	}
	return ganhas
}

// RenovarVarias estende as posses cujo rangeIDid ainda é o dado e cujo
// dono ainda é este nó; devolve as renovadas.
func (b *Banco) RenovarVarias(no string, dur time.Duration,
	tokens map[int]int64) map[int]int64 {
	agora := b.Agora()
	mantidas := map[int]int64{}
	for p, t := range tokens {
		l := &b.posses[p]
		if l.dono == no && l.rangeID == t {
			l.expira = agora.Add(dur)
			mantidas[p] = t
		}
	}
	return mantidas
}

// Soltar devolve a posse, se ainda for a dada.
func (b *Banco) Soltar(no string, p int, token int64) {
	l := &b.posses[p]
	if l.dono == no && l.rangeID == token {
		l.dono, l.expira = "", time.Time{}
		b.avisar(func(o Observador) { o.Soltou(p, no, token) })
	}
}

// livro:fim lease-modelo

// Posse devolve o dono, o rangeIDid e o vencimento da partição p.
func (b *Banco) Posse(p int) (string, int64, time.Time) {
	l := b.posses[p]
	return l.dono, l.rangeID, l.expira
}

func (b *Banco) avisar(f func(Observador)) {
	if b.Observador != nil {
		f(b.Observador)
	}
}

// Entrar é o INSERT ... ON CONFLICT do membership: um reinício zera a
// história do nó.
func (b *Banco) Entrar(no coordinator.NodeID) {
	agora := b.Agora()
	b.membros[no] = &membro{inicio: agora, batida: agora}
}

// Bater grava a batida do nó.
func (b *Banco) Bater(no coordinator.NodeID) {
	if m := b.membros[no]; m != nil {
		m.batida = b.Agora()
	}
}

// Membro é uma linha de cluster_member.
type Membro struct {
	No             coordinator.NodeID
	Inicio, Batida time.Time
}

// Membros lê a tabela, em ordem de nó, com o now() do banco.
func (b *Banco) Membros() ([]Membro, time.Time) {
	var ms []Membro
	for _, no := range slices.Sorted(maps.Keys(b.membros)) {
		m := b.membros[no]
		ms = append(ms, Membro{No: no, Inicio: m.inicio,
			Batida: m.batida})
	}
	return ms, b.Agora()
}

// Liderar é o UPDATE de pgcoord.adquirirOuRenovar: devolve o termo, ou
// 0 se outro nó tem a liderança e ela vale.
func (b *Banco) Liderar(no coordinator.NodeID,
	dur time.Duration) int64 {
	agora := b.Agora()
	if b.lider.no != no && !b.lider.expira.Before(agora) {
		return 0
	}
	if b.lider.no != no {
		b.lider.termo++
	}
	b.lider.no, b.lider.expira = no, agora.Add(dur)
	return b.lider.termo
}

// Redistribuir é a transação de pgcoord.redistribuir: com o termo
// conferido, grava um mapa novo se os vivos não são os donos do mapa
// corrente.
func (b *Banco) Redistribuir(termo int64, vivos []coordinator.NodeID) {
	if b.lider.termo != termo {
		return
	}
	donos := slices.Compact(slices.Sorted(slices.Values(b.mapa.Owners)))
	if slices.Equal(donos, vivos) {
		return
	}
	b.mapa = coordinator.Distribute(vivos, b.mapa)
}

// Mapa devolve o último mapa gravado.
func (b *Banco) Mapa() coordinator.Assignment {
	a := b.mapa
	a.Owners = slices.Clone(a.Owners)
	return a
}
