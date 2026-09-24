#!/usr/bin/env bash
# Cria a branch de uma missão a partir da tag do capítulo.
#   ./scripts/nova-missao.sh 05 lider-perdido cap-26
#
# Depois de plantar o defeito, confira que TestMissao está VERMELHO e que o
# restante da suíte continua verde — é o critério de toda missão (§15.2).
set -euo pipefail

if [[ $# -ne 3 || ! "$1" =~ ^[0-9]{2}$ ]]; then
  echo "uso: $0 NN nome-da-missao cap-NN" >&2
  exit 2
fi
n="$1"; nome="$2"; base="$3"

git rev-parse -q --verify "refs/tags/$base" >/dev/null || { echo "tag $base não existe" >&2; exit 1; }
git checkout -b "missao-$n-$nome" "$base"
echo "branch missao-$n-$nome criada a partir de $base"
