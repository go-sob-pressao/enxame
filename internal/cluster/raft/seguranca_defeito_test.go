//go:build defeito_commit

package raft_test

import "testing"

// livro:inicio figura-8-defeito

// Com a regra ingênua, S1 comita A por contagem em (c) e aplica A no
// índice 2. Em (d), S5 é eleito com votos de quem não tem entrada do
// termo de S1 e sobrescreve o índice 2 com B: uma entrada comitada se
// perdeu, e dois nós aplicaram valores diferentes no mesmo índice.
func TestFigura8SemARegraDeCommit(t *testing.T) {
	c := figura8(t)
	v := c.Violations()
	if len(v) == 0 {
		t.Skip("o roteiro não produziu a violação nesta versão")
	}
	t.Logf("violação: %s", v[0])
}

// livro:fim figura-8-defeito
