#!/usr/bin/env bash
# Cria a tag do capítulo concluído. Nenhum capítulo é publicado com o
# repositório quebrado (Plano Mestre, §16.4).
#   ./scripts/novo-capitulo.sh 07 "Vazamento de goroutine"
set -euo pipefail

if [[ $# -ne 2 || ! "$1" =~ ^[0-9]{2}$ ]]; then
  echo "uso: $0 NN \"Título do capítulo\"   (NN com dois dígitos, 00 a 33)" >&2
  exit 2
fi
n="$1"; titulo="$2"; tag="cap-$n"

if [[ -n "$(git status --porcelain)" ]]; then
  echo "árvore de trabalho suja: faça commit antes de marcar o capítulo" >&2
  exit 1
fi
forcar=""
if git rev-parse -q --verify "refs/tags/$tag" >/dev/null; then
  if [[ -f .tags-publicadas ]]; then
    echo "a tag $tag já existe e foi publicada: correções viram $tag.k (Plano Mestre §16.6)" >&2
    exit 1
  fi
  forcar="-f"   # antes da publicação, a tag é provisória e acompanha o capítulo
fi

make check
git tag $forcar -a "$tag" -m "Capítulo $n — $titulo"
echo "tag $tag criada. publique com: git push origin $tag"
