// Package memory — implementação em memória do Store, com a mesma
// semântica transacional.
//
// Usada pelos testes unitários e pela simulação determinística. Precisa
// reproduzir fielmente isolamento e fencing: se divergir do Postgres, a
// simulação aprova um sistema que falha no banco real. A suíte de
// contrato existe para impedir essa divergência.
//
// Camada: internal/store
// Introduzido no livro: Cap. 12 e Cap. 27 — ver docs/mapa-capitulos.md
package memory
