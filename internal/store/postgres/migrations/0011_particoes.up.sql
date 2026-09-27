-- Partições e ordem por chave (Capítulo 26; ADR-010).
--
-- Com os jobs espalhados pelas 512 partições, a busca do modo biblioteca
-- — que não tem dono de partição e pede de todas — precisa de um índice
-- sem partition_id na frente.
CREATE INDEX job_busca_geral ON job (queue, priority, scheduled_at, job_id)
    WHERE state = 'available';

-- A cabeça de cada chave de ordem: o job mais antigo ainda não final.
-- Só ele pode ser reservado; os outros da chave esperam.
CREATE INDEX job_fila_da_chave ON job (namespace, ordering_key, job_id)
    WHERE ordering_key IS NOT NULL
      AND state NOT IN ('completed', 'discarded', 'cancelled');
