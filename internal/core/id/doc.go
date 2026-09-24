// Package id — identificadores de domínio e a função de
// particionamento.
//
// JobID, RunID, MessageID, PartitionID e a chave de idempotência
// derivada. Nenhum identificador é gerado aqui: a geração (uuid.NewV7)
// é injetada pela borda, para que o domínio permaneça determinístico.
//
// Camada: internal/core Introduzido no livro: Cap. 2 (tipos) e Cap. 16
// (chave de idempotência) — ver docs/mapa-capitulos.md
package id
