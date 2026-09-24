package id

import (
	"fmt"
	"uuid"
)

// JobID identifica um job. É um UUIDv7, ordenável no tempo; quem gera é
// a borda (uuid.NewV7), nunca o domínio.
type JobID uuid.UUID

// String devolve a forma canônica do UUID.
func (j JobID) String() string { return uuid.UUID(j).String() }

// IsZero informa se o identificador não foi preenchido.
func (j JobID) IsZero() bool { return j == JobID{} }

// ParseJobID interpreta a forma textual de um JobID.
func ParseJobID(s string) (JobID, error) {
	u, err := uuid.Parse(s)
	if err != nil {
		return JobID{}, fmt.Errorf("job id %q: %w", s, err)
	}
	return JobID(u), nil
}
