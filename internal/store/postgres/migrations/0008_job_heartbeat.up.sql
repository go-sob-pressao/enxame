-- Batimento de jobs longos (Capítulo 18). Expand: coluna nova, opcional;
-- o resgate passa a medir o prazo a partir do último sinal de vida.
ALTER TABLE job ADD COLUMN heartbeat_at TIMESTAMPTZ;
