package store

import "errors"

// Erros que todas as implementações devolvem, e que a suíte de contrato
// exige com errors.Is.
var (
	// ErrNotFound: o job não existe.
	ErrNotFound = errors.New("job não encontrado")
	// ErrDuplicate: outro job do namespace, nem descartado nem
	// cancelado, já usa a mesma unique_key — ou o job_id já existe.
	ErrDuplicate = errors.New("job duplicado")
	// ErrConflict: a versão gravada não é a esperada; alguém escreveu
	// antes (lock otimista).
	ErrConflict = errors.New("versão desatualizada")
)
