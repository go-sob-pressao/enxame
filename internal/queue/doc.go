// Package queue — filas de jobs: busca, long-poll e limites de
// concorrência.
//
// Modo simples: SELECT … FOR UPDATE SKIP LOCKED. Modo cluster: busca
// apenas nas partições que o nó possui. O long-poll é o estudo de caso
// do select de três vias do Capítulo 5.
//
// Camada: internal/queue Introduzido no livro: Cap. 5, Cap. 6 e Cap. 21
// — ver docs/mapa-capitulos.md
package queue
