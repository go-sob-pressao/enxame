// Package store — contrato de persistência: a transação atômica e as
// leituras do motor.
//
// Interfaces declaradas pelo consumidor (engine, queue), implementadas
// em memory, sqlite e postgres — todas aprovadas pela mesma suíte de
// contrato.
//
// Camada: internal/store Introduzido no livro: Cap. 12 e Caps. 14 a 17
// — ver docs/mapa-capitulos.md
package store
