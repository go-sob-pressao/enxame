#!/usr/bin/env bash
# Recria as tags provisórias de capítulo a partir do histórico linear de main.
# Só vale ANTES da publicação do livro (Plano Mestre, §16.6): depois dela, as
# tags são imutáveis e correções viram cap-NN.k em manutencao/cap-NN.
#
# Cada commit carrega o trailer "Capitulo: NN". A tag cap-NN aponta para o
# ÚLTIMO commit com aquele trailer. Tags intermediárias vêm do trailer
# "Tag: cap-NN-<etapa>" e apontam para o commit que o declara.
#
#   ./scripts/retag.sh              recria as tags
#   ./scripts/retag.sh --verificar  recria e roda `make check` em cada tag
set -euo pipefail

if [[ -f .tags-publicadas ]]; then
  echo "as tags já foram publicadas (.tags-publicadas existe): use cap-NN.k (§16.6)" >&2
  exit 1
fi
if [[ -n "$(git status --porcelain)" ]]; then
  echo "árvore de trabalho suja: faça commit antes de recriar as tags" >&2
  exit 1
fi

verificar=false
[[ "${1:-}" == "--verificar" ]] && verificar=true

# tag<TAB>commit, uma linha por tag; para cap-NN vale o último commit do capítulo.
git log --reverse --format='%H%x09%(trailers:key=Capitulo,valueonly,separator=)%x09%(trailers:key=Tag,valueonly,separator=)' main |
  awk -F'\t' '
    $2 != "" { ultimo["cap-" $2] = $1 }
    $3 != "" { ultimo[$3] = $1 }
    END { for (t in ultimo) print t "\t" ultimo[t] }' |
  sort |
  while IFS=$'\t' read -r tag commit; do
    msg="$(git tag -l --format='%(contents:subject)' "$tag")"
    [[ -z "$msg" ]] && msg="Capítulo ${tag#cap-}"
    git tag -f -a "$tag" -m "$msg" "$commit" >/dev/null
    echo "$tag -> ${commit:0:12}"
  done

if $verificar; then
  atual="$(git rev-parse --abbrev-ref HEAD)"
  for tag in $(git tag -l 'cap-*' | sort); do
    echo "== $tag"
    git checkout -q "$tag"
    make check
  done
  git checkout -q "$atual"
fi
