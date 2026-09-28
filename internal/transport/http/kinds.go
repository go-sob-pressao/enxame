package http

import (
	"regexp"
	"strings"
)

// livro:inicio missao-06

// kindPermitido diz se o kind está na lista de kinds que a API aceita.
// A lista vem da configuração (-kinds); vazia, aceita qualquer um.
func kindPermitido(kinds []string, kind string) bool {
	if len(kinds) == 0 {
		return true
	}
	partes := make([]string, len(kinds))
	for i, k := range kinds {
		partes[i] = regexp.QuoteMeta(k)
	}
	re := regexp.MustCompile("^(" + strings.Join(partes, "|") + ")$")
	return re.MatchString(kind)
}

// livro:fim missao-06
