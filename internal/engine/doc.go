// Package engine — o motor: o que decide quando e onde cada job roda.
//
// Declara as interfaces de que precisa (Store) e as consome; as
// implementações ficam em internal/store e nunca importam este pacote.
//
// Camada: internal/engine
// Introduzido no livro: Cap. 12 (Store) e Caps. 14 a 17 — ver
// docs/mapa-capitulos.md
package engine
