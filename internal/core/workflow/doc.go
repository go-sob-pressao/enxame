// Package workflow — modelo de workflow por passos memoizados e suas
// regras de determinismo.
//
// Um run é uma sequência de passos nomeados. No replay, cada passo já
// concluído devolve o resultado gravado em vez de executar de novo. A
// divergência entre a sequência do código e a do histórico gera
// NonDeterministicError. Ver ADR-002.
//
// Camada: internal/core Introduzido no livro: Cap. 14 (histórico) e
// Cap. 17 (versionamento) — ver docs/mapa-capitulos.md
package workflow
