// Package sharding — algoritmo de atribuição e rebalanceamento de
// partições entre nós.
//
// A função de partição em si (fnv1a da chave % NumPartitions) é domínio
// e mora em internal/core/id. Aqui fica o que depende do cluster:
// calcular a nova atribuição minimizando movimentação e conduzir drenar
// → liberar → adquirir sem perder job. Ver ADR-010.
//
// Camada: internal/cluster
// Introduzido no livro: Cap. 26 — ver docs/mapa-capitulos.md
package sharding
