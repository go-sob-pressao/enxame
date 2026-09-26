package coordinator

import (
	"context"
	"slices"
	"testing"
	"time"
)

// GrupoDeMembros é o que a suíte de membership precisa: um cluster de
// nós, a visão de cada um, e a capacidade de parar um nó e de fazê-lo
// voltar.
type GrupoDeMembros interface {
	Membros() []NodeID
	Visao(id NodeID) Membership
	Parar(id NodeID)
	Voltar(id NodeID)
}

// Grupo é o que a suíte completa precisa de uma implementação: um
// cluster de coordenadores, um por nó, e a capacidade de derrubar nós.
type Grupo interface {
	Membros() []NodeID
	Coordenador(id NodeID) Coordinator
	Parar(id NodeID)
}

// livro:inicio conformidade-membros

// ConformidadeMembros é a parte da suíte que só depende de saber quem
// está vivo. prazo é quanto esperar pela convergência depois de cada
// mudança: a suíte não pergunta quando cada nó percebe, só se todos
// acabam percebendo.
func ConformidadeMembros(
	t *testing.T,
	novo func(t *testing.T, n int) GrupoDeMembros,
	prazo time.Duration,
) {
	t.Run("todos se veem", func(t *testing.T) {
		g := novo(t, 3)
		esperarMembros(t, g, g.Membros(), g.Membros(), prazo)
	})
	t.Run("quem para sai da visão de todos", func(t *testing.T) {
		g := novo(t, 3)
		esperarMembros(t, g, g.Membros(), g.Membros(), prazo)
		vitima := g.Membros()[0]
		g.Parar(vitima)
		vivos := remover(g.Membros(), vitima)
		esperarMembros(t, g, vivos, vivos, prazo)
	})
	t.Run("quem volta, volta para todos", func(t *testing.T) {
		g := novo(t, 3)
		esperarMembros(t, g, g.Membros(), g.Membros(), prazo)
		vitima := g.Membros()[0]
		g.Parar(vitima)
		vivos := remover(g.Membros(), vitima)
		esperarMembros(t, g, vivos, vivos, prazo)
		g.Voltar(vitima)
		esperarMembros(t, g, g.Membros(), g.Membros(), prazo)
	})
}

// livro:fim conformidade-membros

// esperarMembros espera até cada nó de quem ver exatamente esperados.
func esperarMembros(
	t *testing.T,
	g GrupoDeMembros,
	quem, esperados []NodeID,
	prazo time.Duration,
) {
	t.Helper()
	var visto []NodeID
	ok := eventualmente(prazo, func() bool {
		for _, id := range quem {
			m, err := g.Visao(id).Members(context.Background())
			if err != nil || !slices.Equal(m, esperados) {
				visto = m
				return false
			}
		}
		return true
	})
	if !ok {
		t.Fatalf("membros não convergiram para %v em %s; visto %v",
			esperados, prazo, visto)
	}
}

// livro:inicio conformidade

// Conformidade é a suíte que toda implementação de Coordinator precisa
// passar — a mesma para pgcoord e raftcoord (ADR-003); escrita no
// Capítulo 23, antes de haver implementação que a rode. prazo é quanto
// esperar pela convergência depois de cada mudança.
func Conformidade(
	t *testing.T,
	novo func(t *testing.T, n int) Grupo,
	prazo time.Duration,
) {
	t.Run("um líder, reconhecido por todos", func(t *testing.T) {
		g := novo(t, 3)
		lider := esperarLider(t, g, g.Membros(), prazo)
		if !slices.Contains(g.Membros(), lider) {
			t.Fatalf("líder %q não é membro", lider)
		}
	})
	t.Run("mapa completo e balanceado", func(t *testing.T) {
		g := novo(t, 3)
		a := esperarMapa(t, g, g.Membros(), prazo)
		conferirMapa(t, a, g.Membros())
	})
	t.Run(
		"nó cai, partições dele vão para os vivos",
		func(t *testing.T) {
			g := novo(t, 3)
			lider := esperarLider(t, g, g.Membros(), prazo)
			antes := esperarMapa(t, g, g.Membros(), prazo)
			vitima := outro(g.Membros(), lider)
			g.Parar(vitima)
			vivos := remover(g.Membros(), vitima)
			depois := esperarMapa(t, g, vivos, prazo)
			conferirMapa(t, depois, vivos)
			if depois.Epoch <= antes.Epoch {
				t.Fatalf(
					"época não avançou: %d → %d",
					antes.Epoch,
					depois.Epoch,
				)
			}
		},
	)
	t.Run("líder cai, outro assume", func(t *testing.T) {
		g := novo(t, 3)
		antigo := esperarLider(t, g, g.Membros(), prazo)
		g.Parar(antigo)
		vivos := remover(g.Membros(), antigo)
		novoLider := esperarLider(t, g, vivos, prazo)
		if novoLider == antigo {
			t.Fatalf("o líder caído %q continua líder", antigo)
		}
		conferirMapa(t, esperarMapa(t, g, vivos, prazo), vivos)
	})
}

// livro:fim conformidade

// esperarLider espera até todos os nós de ids reconhecerem o mesmo
// líder.
func esperarLider(
	t *testing.T,
	g Grupo,
	ids []NodeID,
	prazo time.Duration,
) NodeID {
	t.Helper()
	var ultimo []NodeID
	ok := eventualmente(prazo, func() bool {
		ultimo = ultimo[:0]
		for _, id := range ids {
			l, err := g.Coordenador(id).Leader(context.Background())
			if err != nil || l == "" {
				return false
			}
			ultimo = append(ultimo, l)
		}
		return len(slices.Compact(slices.Clone(ultimo))) == 1 &&
			slices.Contains(ids, ultimo[0])
	})
	if !ok {
		t.Fatalf("sem líder comum em %s: %v", prazo, ultimo)
	}
	return ultimo[0]
}

// esperarMapa espera até todos os nós de ids verem o mesmo mapa, cujos
// donos são exatamente os nós de ids.
func esperarMapa(
	t *testing.T,
	g Grupo,
	ids []NodeID,
	prazo time.Duration,
) Assignment {
	t.Helper()
	var visto Assignment
	ok := eventualmente(prazo, func() bool {
		var epoca uint64
		for i, id := range ids {
			a, err := g.Coordenador(id).Assignment(context.Background())
			if err != nil || a.Epoch == 0 ||
				len(a.Owners) != NumPartitions {
				return false
			}
			if i > 0 && a.Epoch != epoca {
				return false
			}
			epoca, visto = a.Epoch, a
		}
		// o mapa tem exatamente os nós de ids como donos: nem um a mais
		// (quem caiu), nem um a menos (quem acabou de chegar)
		donos := slices.Compact(
			slices.Sorted(slices.Values(visto.Owners)),
		)
		return slices.Equal(donos, slices.Sorted(slices.Values(ids)))
	})
	if !ok {
		t.Fatalf(
			"mapa não convergiu para %v em %s (época vista %d)",
			ids,
			prazo,
			visto.Epoch,
		)
	}
	return visto
}

func conferirMapa(t *testing.T, a Assignment, ids []NodeID) {
	t.Helper()
	c := map[NodeID]int{}
	for _, o := range a.Owners {
		c[o]++
	}
	menor, maior := NumPartitions, 0
	for _, id := range ids {
		menor, maior = min(menor, c[id]), max(maior, c[id])
	}
	if maior-menor > 1 {
		t.Fatalf("mapa desbalanceado: %v", c)
	}
}

func eventualmente(prazo time.Duration, cond func() bool) bool {
	limite := time.After(prazo)
	for {
		if cond() {
			return true
		}
		select {
		case <-limite:
			return false
		case <-time.After(10 * time.Millisecond):
		}
	}
}

func outro(ids []NodeID, exceto NodeID) NodeID {
	for _, id := range ids {
		if id != exceto {
			return id
		}
	}
	return ""
}

func remover(ids []NodeID, x NodeID) []NodeID {
	return slices.DeleteFunc(
		slices.Clone(ids),
		func(id NodeID) bool { return id == x },
	)
}
