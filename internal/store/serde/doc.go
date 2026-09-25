// Package serde — serialização de argumentos de job e payloads.
//
// Canonical e Equal comparam JSON número a número. Tanto o
// encoding/json quanto o encoding/json/v2 decodificam números em
// float64 quando o destino é any, e perdem os dígitos de inteiros
// acima de 2^53; aqui o decoder usa UseNumber.
//
// Camada: internal/store
// Introduzido no livro: Cap. 13 e Cap. 17 — ver docs/mapa-capitulos.md
package serde
