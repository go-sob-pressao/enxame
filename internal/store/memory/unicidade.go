package memory

import (
	"fmt"

	"github.com/go-sob-pressao/enxame/internal/core/job"
	"github.com/go-sob-pressao/enxame/internal/store"
)

// livro:inicio unicidade

// verificarUnicidade reproduz o índice único parcial do Postgres:
//
//	CREATE UNIQUE INDEX job_unico ON job (namespace, unique_key)
//	    WHERE unique_key IS NOT NULL
//	      AND state NOT IN ('discarded', 'cancelled');
//
// Um job descartado ou cancelado libera a chave; um concluído, não. Um
// fake que só olhasse a chave, ou que a ignorasse, aprovaria no teste o
// que o banco recusa em produção. Chamar com s.mu travado.
func (s *Store) verificarUnicidade(j job.Job) error {
	if !ocupaChave(j) {
		return nil
	}
	for _, r := range s.jobs {
		o := r.job
		if o.ID != j.ID && ocupaChave(o) &&
			o.Namespace == j.Namespace && o.UniqueKey == j.UniqueKey {
			return fmt.Errorf(
				"%w: unique_key %q em uso pelo job %s",
				store.ErrDuplicate,
				j.UniqueKey,
				o.ID,
			)
		}
	}
	return nil
}

// ocupaChave diz se o job entra no índice parcial.
func ocupaChave(j job.Job) bool {
	return j.UniqueKey != "" &&
		j.State != job.StateDiscarded && j.State != job.StateCancelled
}

// livro:fim unicidade
