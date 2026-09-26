-- A volta perde só a hora do último batimento, que é derivável do
-- histórico (job_event, tipo 5).
ALTER TABLE job DROP COLUMN heartbeat_at;
