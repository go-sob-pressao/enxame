#!/usr/bin/env bash
# Deixa todo o código Go dentro de 72 colunas (tabulação = 4), o limite das
# listagens do livro (Plano Mestre §16.7): golines para o código,
# scripts/colunas.py para os comentários, gofmt no fim.
#   scripts/colunas.sh            reformata a árvore
#   scripts/colunas.sh --checar   só lista as linhas acima do limite
set -euo pipefail
raiz="$(git rev-parse --show-toplevel)"
cd "$raiz"
golines="${GOLINES:-$(go env GOPATH)/bin/golines}"

arquivos() { find . -name '*.go' -not -path './.git/*' -not -path './tools/archcheck/testdata/*' -print0; }

if [[ "${1:-}" == "--checar" ]]; then
  arquivos | xargs -0 python3 -c '
import sys
n = 0
for p in sys.argv[1:]:
    for i, l in enumerate(open(p, encoding="utf-8"), 1):
        t = l.rstrip("\n").expandtabs(4)
        if len(t) > 72:
            print(f"{p}:{i}: {len(t)} colunas")
            n += 1
sys.exit(1 if n else 0)
'
  exit
fi

arquivos | xargs -0 "$golines" -m 72 -t 4 --no-shorten-comments -w
arquivos | xargs -0 python3 "$raiz/scripts/colunas.py"
arquivos | xargs -0 gofmt -w
