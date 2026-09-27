# ADR 0010 — Número de partições fixo (512)

- **Status:** aceita
- **Capítulo:** 26
- **Data:** 2026-09-27

## Contexto

O trabalho do Enxame é dividido entre os nós por partição: cada job cai
numa partição pela chave de ordem (ou pelo próprio id), e cada partição
tem um dono por vez, com fencing token no banco (ADR-005). A função que
leva a chave à partição precisa dar a mesma resposta em todo nó, em toda
versão, para todo job já gravado — a coluna `partition_id` de cada job
foi calculada no insert.

## Opções consideradas

1. **Número de partições fixo**, grande em relação ao número de nós
   (512), com a função `fnv1a(chave) % 512`.
2. **Hash consistente** sobre um anel de nós, sem partições: cada chave
   vai para o nó seguinte no anel.
3. **Reparticionamento dinâmico**: dividir e juntar partições conforme a
   carga.

## Decisão

Opção 1. As partições são a unidade de posse e de movimento; o mapa
partição→nó, e só ele, muda quando o cluster muda. Com 512 partições e
até algumas dezenas de nós, o desequilíbrio entre nós fica abaixo de
uma partição por nó, e o rebalanceamento move só as partições de quem
entrou ou saiu.

## Consequências

- A função e o número não mudam nunca: `TestParticaoEstavel` guarda o
  valor de uma chave calculado fora do Go. Mudar exigiria migrar a
  coluna `partition_id` de todos os jobs.
- O hash consistente não compra nada aqui: o que ele economiza em
  movimento o mapa de partições já economiza, e ele não tem unidade de
  posse onde pendurar o fencing.
- Uma chave muito quente continua numa partição só — e num nó só. O
  particionamento espalha chaves, não a carga de uma chave (a partição
  quente do Capítulo 26).
- 512 é teto para o paralelismo de posse: um cluster com mais de 512
  nós teria nós sem partição.
