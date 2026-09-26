-- workflow.Version (Capítulo 17): o marcador de versão é um passo do
-- histórico, de um tipo novo. Expand: o CHECK passa a aceitar 'version'.
ALTER TABLE workflow_step DROP CONSTRAINT workflow_step_step_kind_check;
ALTER TABLE workflow_step ADD CONSTRAINT workflow_step_step_kind_check
    CHECK (step_kind IN ('call', 'sleep', 'signal', 'side_effect',
                         'version'));
