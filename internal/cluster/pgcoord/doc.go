// Package pgcoord — coordenação sobre PostgreSQL: lease, fencing e
// eleição por linha de controle.
//
// A implementação padrão de produção: usa o banco que o sistema já
// possui.
//
// Camada: internal/cluster
// Introduzido no livro: Cap. 24 — ver docs/mapa-capitulos.md
package pgcoord
