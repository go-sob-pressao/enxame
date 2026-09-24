// Package partition — propriedade de partição: lease, fencing token
// (range_id) e ciclo de vida.
//
// Um nó só escreve em uma partição se o range_id que carrega for o
// vigente, verificado com SELECT … FOR SHARE dentro da própria
// transação. Ver ADR-005.
//
// Camada: internal/engine
// Introduzido no livro: Cap. 24 — ver docs/mapa-capitulos.md
package partition
