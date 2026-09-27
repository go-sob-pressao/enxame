//go:build simulation

package simulation

import (
	"context"
	"flag"
	"fmt"
	"hash"
	"hash/fnv"
	"log/slog"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/go-sob-pressao/enxame/internal/cluster/coordinator"
	"github.com/go-sob-pressao/enxame/internal/cluster/membership"
	"github.com/go-sob-pressao/enxame/internal/cluster/sharding"
	"github.com/go-sob-pressao/enxame/internal/engine/partition"
	"github.com/go-sob-pressao/enxame/internal/simulation/banco"
	"github.com/go-sob-pressao/enxame/internal/simulation/clock"
	"github.com/go-sob-pressao/enxame/internal/simulation/invariant"
	"github.com/go-sob-pressao/enxame/internal/simulation/scheduler"
	"github.com/go-sob-pressao/enxame/internal/simulation/storefault"
)

// Os tempos do cenário: os do enxamed, com o resgate encurtado de um
// minuto para cinco segundos, para caber no cenário.
const (
	batimento   = 500 * time.Millisecond // membership e pgcoord
	passo       = 200 * time.Millisecond // rebalanceador (Intervalo/5)
	leaseLider  = 3 * time.Second
	leaseMotor  = 3 * time.Second
	semBatida   = 5 * time.Second // resgate
	caos        = 40 * time.Second
	fimCenario  = 100 * time.Second
	maxEmCurso  = 4
	chavesCarga = 20
)

// mundo é um cenário: o banco, os nós, o escalonador e o que se
// observa.
type mundo struct {
	s      *scheduler.Scheduler
	b      *banco.Banco
	nos    []*no
	inv    *invariant.Enxame
	rastro hash.Hash64
	diario []string
	base   time.Time
}

// no é um enxamed simulado: membership, coordenador, rebalanceador,
// worker e motor, cada um conduzido pelo escalonador.
type no struct {
	m       *mundo
	id      coordinator.NodeID
	geracao int // muda a cada reinício: eventos da vida anterior morrem
	vivo    bool
	pausa   time.Duration // parado até este instante virtual
	con     *storefault.Conexao

	outros map[coordinator.NodeID]*observado
	agora  time.Time // now() do banco na última leitura

	mapa   coordinator.Assignment
	posses *partition.Posses
	reb    *sharding.Rebalanceador
	rodam  int // tentativas em curso neste nó
}

type observado struct {
	det            membership.Detector
	inicio, ultima time.Time
}

func (m *mundo) Now() time.Time { return m.base.Add(m.s.Now()) }

func (m *mundo) NewTimer(time.Duration) clock.Timer {
	panic("a simulação não usa timers")
}

func (m *mundo) anotar(f string, a ...any) {
	m.diario = append(m.diario, fmt.Sprintf("%7.3fs ",
		m.s.Now().Seconds())+fmt.Sprintf(f, a...))
}

// O mundo observa o banco para as invariantes e para o rastro.
func (m *mundo) Reservou(j banco.Job, no string, token int64) {
	fmt.Fprintf(m.rastro, "r%d:%d:%s:%d;", j.ID, j.Tentativa, no, token)
	m.inv.Reservou(j, no, token)
	if j.Particao == *particao {
		m.anotar("%s reserva o job %d (tentativa %d)", no, j.ID,
			j.Tentativa)
	}
}

func (m *mundo) Concluiu(j banco.Job, no string) {
	fmt.Fprintf(m.rastro, "c%d:%s;", j.ID, no)
	m.inv.Concluiu(j, no)
	if j.Particao == *particao {
		m.anotar("%s conclui o job %d", no, j.ID)
	}
}

func (m *mundo) Adquiriu(p int, no string, token int64) {
	fmt.Fprintf(m.rastro, "a%d:%s:%d;", p, no, token)
	if p == *particao {
		m.anotar("%s adquire a partição %d (token %d)", no, p, token)
	}
}

func (m *mundo) Soltou(p int, no string, token int64) {
	if p == *particao {
		m.anotar("%s solta a partição %d (token %d)", no, p, token)
	}
}

var particao = flag.Int("particao", -1,
	"anota no diário as posses desta partição")

// livro:inicio cada

// cada executa fn neste nó a cada periodo, a partir de um atraso
// sorteado. O evento de um nó parado é adiado para quando ele voltar;
// o de um nó morto, ou de uma vida anterior, simplesmente não acontece.
func (n *no) cada(periodo time.Duration, fn func()) {
	g := n.geracao
	var vez func()
	vez = func() {
		if !n.vivo || n.geracao != g {
			return
		}
		if agora := n.m.s.Now(); agora < n.pausa {
			n.m.s.After(n.pausa-agora, vez)
			return
		}
		fn()
		n.m.s.After(periodo, vez)
	}
	n.m.s.After(time.Duration(n.m.s.Rand().Int64N(int64(periodo))), vez)
}

// livro:fim cada

// depois executa fn uma vez, daqui a d, com as mesmas regras de cada.
func (n *no) depois(d time.Duration, fn func()) {
	g := n.geracao
	var vez func()
	vez = func() {
		if !n.vivo || n.geracao != g {
			return
		}
		if agora := n.m.s.Now(); agora < n.pausa {
			n.m.s.After(n.pausa-agora, vez)
			return
		}
		fn()
	}
	n.m.s.After(d, vez)
}

// iniciar é o processo enxamed subindo: estado em memória zerado.
func (n *no) iniciar() {
	m := n.m
	n.geracao++
	n.vivo, n.rodam = true, 0
	n.outros = map[coordinator.NodeID]*observado{}
	n.mapa = coordinator.Assignment{}
	n.posses = &partition.Posses{}
	n.reb = &sharding.Rebalanceador{No: n.id, Coord: n,
		Lease: leaseSim{n}, Posses: n.posses, Relogio: m,
		EmCurso: func(_ context.Context, ps []int) (map[int]int,
			error) {
			var r map[int]int
			err := n.con.Fazer(func() { r = m.b.EmCurso(ps) })
			return r, err
		},
		Intervalo: time.Second, Drenagem: 30 * time.Second,
		Espera: 2 * time.Second,
		Log:    slog.New(&diarioHandler{m: m, no: n.id})}
	_ = n.con.Fazer(func() { m.b.Entrar(n.id) })
	n.cada(batimento, n.membership)
	n.cada(batimento, n.coordenar)
	n.cada(passo, func() { n.reb.Passo(context.Background()) })
	n.cada(50*time.Millisecond, n.reservar)
	n.cada(time.Second, n.resgatar)
}

// membership é o ciclo de membership.Membro sobre o banco simulado.
func (n *no) membership() {
	var ms []banco.Membro
	err := n.con.Fazer(func() {
		n.m.b.Bater(n.id)
		ms, n.agora = n.m.b.Membros()
	})
	if err != nil {
		return
	}
	for _, x := range ms {
		if x.No == n.id {
			continue
		}
		o := n.outros[x.No]
		if o == nil || !o.inicio.Equal(x.Inicio) {
			o = &observado{det: membership.NovoPhi(batimento),
				inicio: x.Inicio}
			n.outros[x.No] = o
		}
		if !o.ultima.Equal(x.Batida) {
			o.det.Batida(x.Batida)
			o.ultima = x.Batida
		}
	}
}

// Members é o de membership.Membro: este nó e os que ele não dá por
// mortos.
func (n *no) Members(context.Context) ([]coordinator.NodeID, error) {
	ids := []coordinator.NodeID{n.id}
	for id, o := range n.outros {
		if o.det.Estado(n.agora) != membership.Morto {
			ids = append(ids, id)
		}
	}
	slices.Sort(ids)
	return ids, nil
}

// coordenar é o ciclo do pgcoord: lease de liderança, redistribuição no
// líder, releitura do mapa. Cada instrução pode falhar sozinha.
func (n *no) coordenar() {
	b := n.m.b
	var termo int64
	if n.con.Fazer(func() { termo = b.Liderar(n.id, leaseLider) }) != nil {
		return
	}
	if termo > 0 {
		vivos, _ := n.Members(context.Background())
		antes := b.Mapa().Epoch
		if n.con.Fazer(func() { b.Redistribuir(termo, vivos) }) != nil {
			return
		}
		if a := b.Mapa(); a.Epoch != antes {
			n.m.anotar("%s (líder, termo %d) grava o mapa %d: %v%s",
				n.id, termo, a.Epoch, vivos, n.m.donoNoMapa(a))
		}
	}
	_ = n.con.Fazer(func() { n.mapa = b.Mapa() })
}

// Leader, Assignment e Close completam o coordinator.Coordinator.
func (n *no) Leader(context.Context) (coordinator.NodeID, error) {
	return "", nil
}

func (n *no) Assignment(context.Context) (coordinator.Assignment,
	error) {
	return n.mapa, nil
}

func (n *no) Close() error { return nil }

// reservar é o poller do pool: uma reserva por vez, até maxEmCurso.
func (n *no) reservar() {
	if n.rodam >= maxEmCurso {
		return
	}
	var (
		j       banco.Job
		achou   bool
		perdida = -1
	)
	tokens := n.posses.Tokens()
	if len(tokens) == 0 {
		return
	}
	if n.con.Fazer(func() {
		j, achou, perdida = n.m.b.Reservar(string(n.id), tokens)
	}) != nil {
		return
	}
	if perdida >= 0 {
		n.posses.Largar(perdida) // Dono.Perdeu
	}
	if !achou {
		return
	}
	n.rodam++
	r := n.m.s.Rand()
	d := time.Duration(20+r.IntN(380)) * time.Millisecond
	if r.IntN(20) == 0 {
		d = time.Duration(2+r.IntN(5)) * time.Second // um job lento
	}
	for t := time.Second; t < d; t += time.Second {
		n.depois(t, func() {
			_ = n.con.Fazer(func() { n.m.b.BaterJob(j.ID, j.Tentativa) })
		})
	}
	n.depois(d, func() {
		n.rodam--
		_ = n.con.Fazer(func() {
			n.m.b.Concluir(string(n.id), j.ID, j.Tentativa)
		})
	})
}

// resgatar é o Rescue do motor, nas partições ativas do nó.
func (n *no) resgatar() {
	tokens := n.posses.Tokens()
	b := n.m.b
	_ = n.con.Fazer(func() {
		b.Resgatar(tokens, b.Agora().Add(-semBatida))
	})
}

// leaseSim é o partition.Lease sobre o banco simulado.
type leaseSim struct{ n *no }

func (l leaseSim) AdquirirVarias(_ context.Context,
	ps []int) (map[int]int64, error) {
	var r map[int]int64
	err := l.n.con.Fazer(func() {
		r = l.n.m.b.AdquirirVarias(string(l.n.id), leaseMotor, ps)
	})
	return r, err
}

func (l leaseSim) RenovarVarias(_ context.Context,
	t map[int]int64) (map[int]int64, error) {
	var r map[int]int64
	err := l.n.con.Fazer(func() {
		r = l.n.m.b.RenovarVarias(string(l.n.id), leaseMotor, t)
	})
	return r, err
}

func (l leaseSim) Soltar(_ context.Context, p int, token int64) error {
	return l.n.con.Fazer(func() {
		l.n.m.b.Soltar(string(l.n.id), p, token)
	})
}

// livro:inicio cenario-enxame

// cenarioEnxame roda três nós sobre o banco simulado: 40 s de carga e
// falhas sorteadas, depois 60 s de calma, com todos os nós de pé e a
// rede inteira. No fim, todo job tem de ter sido concluído.
func cenarioEnxame(seed uint64) *mundo {
	s := scheduler.New(seed)
	m := &mundo{s: s, inv: invariant.NewEnxame(), rastro: fnv.New64a(),
		base: time.Date(2026, 3, 13, 3, 12, 0, 0, time.UTC)}
	m.b = banco.Novo(m.Now)
	m.b.Observador = m
	for _, id := range []coordinator.NodeID{"no-1", "no-2", "no-3"} {
		n := &no{m: m, id: id, con: &storefault.Conexao{Rand: s.Rand()}}
		m.nos = append(m.nos, n)
		n.iniciar()
	}
	seq := map[string]int{}
	var carga func()
	carga = func() {
		chave := fmt.Sprintf("pedido-%d", s.Rand().IntN(chavesCarga))
		m.b.Enfileirar(chave, seq[chave])
		seq[chave]++
		if s.Now() < caos {
			s.After(100*time.Millisecond, carga)
		}
	}
	s.After(time.Second, carga)
	var falha func()
	falha = func() {
		m.falhar()
		if s.Now() < caos {
			s.After(time.Duration(500+s.Rand().IntN(2500))*
				time.Millisecond, falha)
		}
	}
	s.After(5*time.Second, falha)
	s.After(caos, m.curar)
	s.Run(fimCenario)
	m.inv.Fim(m.b, m.explicar)
	return m
}

// livro:fim cenario-enxame

// livro:inicio falhas-enxame

// falhar sorteia uma falha: um nó parado (uma pausa longa de GC, um
// SIGSTOP), um nó que cai e volta, um nó isolado do banco, ou uma
// conexão que perde instruções e respostas.
func (m *mundo) falhar() {
	if m.s.Now() >= caos {
		return
	}
	r := m.s.Rand()
	n := m.nos[r.IntN(len(m.nos))]
	d := time.Duration(300+r.IntN(5700)) * time.Millisecond
	switch x := r.IntN(100); {
	case x < 30 && n.vivo:
		n.pausa = m.s.Now() + d
		m.anotar("%s parado por %s", n.id, d)
	case x < 45 && n.vivo && m.todosVivos():
		n.vivo = false
		m.anotar("%s caiu; volta em %s", n.id, d)
		m.s.After(d, func() {
			if !n.vivo {
				m.anotar("%s voltou", n.id)
				n.iniciar()
			}
		})
	case x < 65 && !n.con.Cortada:
		n.con.Cortada = true
		m.anotar("%s isolado do banco por %s", n.id, d)
		m.s.After(d, func() {
			n.con.Cortada = false
			m.anotar("%s de volta ao banco", n.id)
		})
	case x < 80 && n.con.Perda == 0:
		n.con.Perda = 0.05
		m.anotar("%s perde 5%% das instruções por %s", n.id, d)
		m.s.After(d, func() { n.con.Perda = 0 })
	}
}

// livro:fim falhas-enxame

func (m *mundo) todosVivos() bool {
	for _, n := range m.nos {
		if !n.vivo {
			return false
		}
	}
	return true
}

func (m *mundo) donoNoMapa(a coordinator.Assignment) string {
	if *particao < 0 {
		return ""
	}
	return fmt.Sprintf("; a partição %d é de %s", *particao,
		a.Owners[*particao])
}

// curar encerra o caos: todo nó de pé, rede inteira.
func (m *mundo) curar() {
	m.anotar("fim das falhas")
	for _, n := range m.nos {
		n.con.Cortada, n.con.Perda = false, 0
		n.pausa = 0
		if !n.vivo {
			n.iniciar()
		}
	}
}

// explicar diz, de um job perdido, o que o mapa e o banco sabem da
// partição dele.
func (m *mundo) explicar(j *banco.Job) string {
	a := m.b.Mapa()
	dono, token, expira := m.b.Posse(j.Particao)
	if dono == "" || expira.Before(m.Now()) {
		dono = "ninguém"
	}
	return fmt.Sprintf("no mapa %d, é de %s; no banco, a posse é de %s "+
		"(token %d)", a.Epoch, a.Owners[j.Particao], dono, token)
}

// diarioHandler leva os logs dos nós para o diário do cenário.
type diarioHandler struct {
	m  *mundo
	no coordinator.NodeID
}

func (h *diarioHandler) Enabled(context.Context, slog.Level) bool {
	return true
}

func (h *diarioHandler) Handle(_ context.Context, r slog.Record) error {
	var b strings.Builder
	r.Attrs(func(a slog.Attr) bool {
		fmt.Fprintf(&b, " %s=%v", a.Key, a.Value)
		return true
	})
	h.m.anotar("%s: %s%s", h.no, r.Message, b.String())
	return nil
}

func (h *diarioHandler) WithAttrs([]slog.Attr) slog.Handler { return h }
func (h *diarioHandler) WithGroup(string) slog.Handler      { return h }

// livro:inicio teste-enxame

func TestSimulacaoEnxame(t *testing.T) {
	inicio := time.Now()
	ss := seeds()
	for _, seed := range ss {
		if v := rodarEnxame(t, seed); len(v) > 0 {
			registrarFalha(t, seed, v)
		}
	}
	t.Logf("%d cenários de %s virtuais em %s", len(ss), fimCenario,
		time.Since(inicio).Round(time.Millisecond))
}

// livro:fim teste-enxame

func rodarEnxame(t *testing.T, seed uint64) (violacoes []string) {
	defer func() {
		if r := recover(); r != nil {
			violacoes = append(violacoes, fmt.Sprintf("pânico: %v", r))
		}
	}()
	m := cenarioEnxame(seed)
	if len(m.inv.Violacoes) > 0 && *seedUnica != 0 {
		t.Logf("diário da seed %d:\n%s", seed,
			strings.Join(m.diario, "\n"))
	}
	return m.inv.Violacoes
}
