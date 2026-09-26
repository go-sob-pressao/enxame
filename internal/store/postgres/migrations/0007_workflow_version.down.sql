-- A volta estreita o CHECK de novo. Se algum run já gravou um marcador
-- de versão, o ALTER falha e nada muda: a volta recusa em vez de apagar
-- histórico.
ALTER TABLE workflow_step DROP CONSTRAINT workflow_step_step_kind_check;
ALTER TABLE workflow_step ADD CONSTRAINT workflow_step_step_kind_check
    CHECK (step_kind IN ('call', 'sleep', 'signal', 'side_effect'));
