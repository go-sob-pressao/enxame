// Package clock — relógio virtual controlado pelo escalonador.
//
// Nasce no Cap. 11 como relógio injetável; testing/synctest cobre o
// teste unitário, e este pacote cobre o que synctest não alcança:
// vários nós com relógios divergentes.
//
// Camada: internal/simulation
// Introduzido no livro: Cap. 11 e Cap. 27 — ver docs/mapa-capitulos.md
package clock
