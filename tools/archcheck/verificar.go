package main

import (
	"cmp"
	"fmt"
	"go/ast"
	"go/types"
	"maps"
	"path/filepath"
	"slices"
	"strings"

	"golang.org/x/tools/go/packages"
)

// Violacao é uma quebra de regra de arquitetura. Pos é arquivo:linha
// quando a violação é uma chamada; vazio quando é um import.
type Violacao struct {
	Pacote string
	Pos    string
	Motivo string
}

func (v Violacao) String() string {
	if v.Pos != "" {
		return v.Pos + ": " + v.Motivo
	}
	return v.Pacote + ": " + v.Motivo
}

// verificar carrega todos os pacotes do módulo em dir e devolve as
// violações em ordem estável.
func verificar(dir string) ([]Violacao, error) {
	cfg := &packages.Config{
		Dir: dir,
		Mode: packages.NeedName | packages.NeedFiles | packages.NeedImports |
			packages.NeedModule | packages.NeedSyntax | packages.NeedTypes | packages.NeedTypesInfo,
	}
	pkgs, err := packages.Load(cfg, "./...")
	if err != nil {
		return nil, fmt.Errorf("carregar pacotes: %w", err)
	}

	var erros []string
	for _, p := range pkgs {
		for _, e := range p.Errors {
			erros = append(erros, e.Error())
		}
	}
	if len(erros) > 0 {
		return nil, fmt.Errorf(
			"pacotes com erro de compilação:\n  %s",
			strings.Join(erros, "\n  "),
		)
	}

	var vs []Violacao
	for _, p := range pkgs {
		if p.Module == nil {
			continue
		}
		rel := relativo(p.PkgPath, p.Module.Path)
		vs = append(vs, violacoesDeCamada(p, rel)...)
		if dentro(rel, camadaPura) {
			vs = append(vs, violacoesDePureza(p, rel)...)
		}
	}

	slices.SortFunc(vs, func(a, b Violacao) int {
		return cmp.Or(
			strings.Compare(a.Pacote, b.Pacote),
			strings.Compare(a.Pos, b.Pos),
			strings.Compare(a.Motivo, b.Motivo),
		)
	})
	return slices.Compact(vs), nil
}

func violacoesDeCamada(p *packages.Package, rel string) []Violacao {
	var vs []Violacao
	for _, caminho := range slices.Sorted(maps.Keys(p.Imports)) {
		imp := p.Imports[caminho]
		if imp.Module == nil || imp.Module.Path != p.Module.Path {
			continue // stdlib e terceiros não são camadas deste módulo
		}
		relImp := relativo(caminho, p.Module.Path)
		for _, r := range regrasDeCamada {
			if !dentro(rel, r.Camada) || dentro(relImp, r.Camada) {
				continue
			}
			for _, proibido := range r.Proibidos {
				if dentro(relImp, proibido) {
					vs = append(vs, Violacao{
						Pacote: rel,
						Motivo: fmt.Sprintf(
							"importa %s (camada %q não pode importar %q)",
							relImp,
							r.Camada,
							proibido,
						),
					})
				}
			}
		}
	}
	return vs
}

func violacoesDePureza(p *packages.Package, rel string) []Violacao {
	var vs []Violacao
	for _, caminho := range slices.Sorted(maps.Keys(p.Imports)) {
		imp := p.Imports[caminho]
		switch {
		case imp.Module != nil && imp.Module.Path != p.Module.Path:
			vs = append(
				vs,
				Violacao{
					Pacote: rel,
					Motivo: fmt.Sprintf(
						"importa a biblioteca de terceiros %s (%s é domínio puro)",
						caminho,
						camadaPura,
					),
				},
			)
		case imp.Module == nil && slices.ContainsFunc(stdlibProibidaNoCore, func(pre string) bool { return dentro(caminho, pre) }):
			vs = append(
				vs,
				Violacao{
					Pacote: rel,
					Motivo: fmt.Sprintf(
						"importa %s, que faz I/O ou introduz não determinismo (%s é domínio puro)",
						caminho,
						camadaPura,
					),
				},
			)
		}
	}

	for _, arq := range p.Syntax {
		ast.Inspect(arq, func(n ast.Node) bool {
			sel, ok := n.(*ast.SelectorExpr)
			if !ok {
				return true
			}
			fn, ok := p.TypesInfo.Uses[sel.Sel].(*types.Func)
			if !ok || fn.Pkg() == nil || fn.Signature().Recv() != nil {
				return true
			}
			for _, f := range funcoesProibidasNoCore {
				if fn.Pkg().Path() == f.Pacote &&
					slices.Contains(f.Nomes, fn.Name()) {
					pos := p.Fset.Position(sel.Pos())
					arquivo, err := filepath.Rel(
						p.Module.Dir,
						pos.Filename,
					)
					if err != nil {
						arquivo = pos.Filename
					}
					vs = append(vs, Violacao{
						Pacote: rel,
						Pos: fmt.Sprintf(
							"%s:%d",
							filepath.ToSlash(arquivo),
							pos.Line,
						),
						Motivo: fmt.Sprintf(
							"usa %s.%s — %s",
							f.Pacote,
							fn.Name(),
							f.Substituto,
						),
					})
				}
			}
			return true
		})
	}
	return vs
}

// relativo devolve o caminho do pacote relativo à raiz do módulo.
func relativo(caminho, modulo string) string {
	if caminho == modulo {
		return "."
	}
	return strings.TrimPrefix(caminho, modulo+"/")
}

// dentro informa se caminho é prefixo ou está abaixo de prefixo,
// respeitando a fronteira de segmento: "internal/core" contém
// "internal/core/job", mas não "internal/corex".
func dentro(caminho, prefixo string) bool {
	return caminho == prefixo || strings.HasPrefix(caminho, prefixo+"/")
}
