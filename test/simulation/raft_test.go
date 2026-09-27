//go:build simulation

// Package simulation — cenários de simulação determinística (Cap. 27).
//
//	go test -tags=simulation ./test/simulation/                      seeds 1..200
//	go test -tags=simulation ./test/simulation/ -args -seed=8371      uma seed
//	go test -tags=simulation ./test/simulation/ -args -random-seeds=2000 -lote=1
package simulation

import (
	"encoding/json"
	"flag"
	"fmt"
	"hash/fnv"
	"math/rand/v2"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/go-sob-pressao/enxame/internal/cluster/raft"
	"github.com/go-sob-pressao/enxame/internal/simulation/invariant"
	"github.com/go-sob-pressao/enxame/internal/simulation/network"
	"github.com/go-sob-pressao/enxame/internal/simulation/scheduler"
)

var (
	seedUnica  = flag.Uint64("seed", 0, "executa só esta seed")
	seedsAleat = flag.Int(
		"random-seeds",
		0,
		"quantas seeds aleatórias varrer",
	)
	lote = flag.Int(
		"lote",
		0,
		"lote da varredura noturna (entra no sorteio das seeds)",
	)
	duracaoVirt = flag.Duration(
		"duracao",
		30*time.Second,
		"tempo virtual por cenário",
	)
)

// livro:inicio cenario-raft

// cenarioRaft roda cinco nós Raft sobre a rede virtual por um tempo
// virtual, com falhas sorteadas, e devolve as violações observadas.
func cenarioRaft(
	seed uint64,
	duracao time.Duration,
) (*invariant.Raft, network.Stats, uint64) {
	s := scheduler.New(seed)
	rastro := fnv.New64a()
	inv := invariant.NewRaft()
	ids := []raft.NodeID{1, 2, 3, 4, 5}
	nos := map[raft.NodeID]*raft.Node{}
	fora := map[raft.NodeID]bool{}
	// livro:inicio cenario-rede

	aplicadas := map[raft.NodeID][]raft.Entry{}

	var rede *network.Network[raft.Message]
	processar := func(id raft.NodeID) {
		r := nos[id].Ready()
		if r.Snapshot != nil {
			var es []raft.Entry
			_ = json.Unmarshal(r.Snapshot.Data, &es)
			aplicadas[id] = nil
			for _, e := range es {
				inv.Aplicada(id, e)
				aplicadas[id] = append(aplicadas[id], e)
			}
		}
		for _, e := range r.Committed {
			inv.Aplicada(id, e)
			aplicadas[id] = append(aplicadas[id], e)
		}
		if st := nos[id].Status(); st.State == raft.Leader {
			inv.Lider(st.Term, id)
		}
		for _, m := range r.Messages {
			rede.Send(int(m.From), int(m.To), m)
		}
	}
	rede = network.New(s, network.Config{
		AtrasoMin: time.Millisecond, AtrasoMax: 20 * time.Millisecond,
		Perda: 0.02, Duplicacao: 0.02,
	}, func(de, para int, m raft.Message) {
		id := raft.NodeID(para)
		if fora[id] {
			return
		}
		fmt.Fprintf(rastro, "%d>%d:%d:%d;", de, para, m.Type, m.Term)
		nos[id].Step(m)
		processar(id)
	})
	// livro:fim cenario-rede

	// livro:inicio cenario-nos

	for _, id := range ids {
		nos[id] = raft.New(raft.Config{
			ID: id, Peers: ids, ElectionTicks: 10, HeartbeatTicks: 2,
			Rand: s.Rand().IntN,
		})
	}
	for _, id := range ordemDosNos(nos) {
		var tick func()
		tick = func() {
			if !fora[id] {
				nos[id].Tick()
				processar(id)
			}
			s.After(10*time.Millisecond, tick)
		}
		s.After(time.Duration(s.Rand().IntN(10))*time.Millisecond, tick)
	}
	// livro:fim cenario-nos

	// livro:inicio cenario-falhas

	proposta := 0
	var falha func()
	falha = func() {
		r := s.Rand()
		alvo := ids[r.IntN(len(ids))]
		switch x := r.IntN(100); {
		case x < 8 && len(fora) < 2:
			fora[alvo] = true
		case x < 16:
			delete(fora, alvo)
		case x < 21:
			a, b := particionar(r, ids)
			rede.Partition(a, b)
		case x < 26:
			rede.Heal()
		case x < 31:
			if !fora[alvo] && len(aplicadas[alvo]) > 0 {
				dados, _ := json.Marshal(aplicadas[alvo])
				ultima := aplicadas[alvo][len(aplicadas[alvo])-1].Index
				_ = nos[alvo].Compact(ultima, dados)
			}
		default:
			for _, id := range ids {
				if !fora[id] && nos[id].Status().State == raft.Leader {
					proposta++
					_, _ = nos[id].Propose(
						fmt.Appendf(nil, "cmd-%d", proposta),
					)
					processar(id)
					break
				}
			}
		}
		s.After(time.Duration(50+r.IntN(200))*time.Millisecond, falha)
	}
	s.After(time.Second, falha)
	s.Run(duracao)

	// livro:fim cenario-falhas

	logs := map[raft.NodeID][]raft.Entry{}
	for _, id := range ids {
		logs[id] = nos[id].Log()
	}
	inv.LogMatching(logs)
	return inv, rede.Stats, rastro.Sum64()
}

// livro:fim cenario-raft

func particionar(r *rand.Rand, ids []raft.NodeID) (a, b []int) {
	for _, id := range ids {
		if r.IntN(2) == 0 {
			a = append(a, int(id))
		} else {
			b = append(b, int(id))
		}
	}
	return a, b
}

func seeds() []uint64 {
	switch {
	case *seedUnica != 0:
		return []uint64{*seedUnica}
	case *seedsAleat > 0:
		r := rand.New(
			rand.NewPCG(uint64(time.Now().UnixNano()), uint64(*lote)),
		)
		out := make([]uint64, *seedsAleat)
		for i := range out {
			out[i] = r.Uint64()
		}
		return out
	default:
		out := make([]uint64, 200)
		for i := range out {
			out[i] = uint64(i + 1)
		}
		return out
	}
}

func TestSimulacaoRaft(t *testing.T) {
	inicio := time.Now()
	ss := seeds()
	for _, seed := range ss {
		if v := rodarSeed(seed); len(v) > 0 {
			registrarFalha(t, seed, v)
		}
	}
	t.Logf(
		"%d cenários de %s virtuais em %s",
		len(ss),
		*duracaoVirt,
		time.Since(inicio).Round(time.Millisecond),
	)
}

// rodarSeed executa um cenário e devolve as violações. Um pânico dentro
// do sistema simulado também é um achado: vira violação, com a seed.
func rodarSeed(seed uint64) (violacoes []string) {
	defer func() {
		if r := recover(); r != nil {
			violacoes = append(violacoes, fmt.Sprintf("pânico: %v", r))
		}
	}()
	inv, _, _ := cenarioRaft(seed, *duracaoVirt)
	return inv.Violacoes
}

func registrarFalha(t *testing.T, seed uint64, v []string) {
	t.Helper()
	_ = os.MkdirAll("failures", 0o755)
	arq := filepath.Join("failures", fmt.Sprintf("seed-%d.txt", seed))
	_ = os.WriteFile(
		arq,
		fmt.Appendf(nil, "seed %d\n%v\n", seed, v),
		0o644,
	)
	t.Errorf(
		"seed %d: %s (reproduza com -args -seed=%d)",
		seed,
		v[0],
		seed,
	)
}
