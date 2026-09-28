-- O contexto de trace do job (W3C traceparent), gravado junto com ele
-- (Cap. 30): o trace de um job, ou de um run inteiro, sobrevive ao
-- banco, a reinícios e a dias de espera entre um passo e outro.
ALTER TABLE job ADD COLUMN trace_parent TEXT;
