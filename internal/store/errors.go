package store

import "errors"

// Erros que todas as implementações devolvem, e que a suíte de contrato
// exige com errors.Is.
var (
	// ErrNotFound: o registro — job, run, endpoint — não existe.
	ErrNotFound = errors.New("não encontrado")
	// ErrDuplicate: outro job do namespace, nem descartado nem
	// cancelado, já usa a mesma unique_key — ou o job_id já existe.
	ErrDuplicate = errors.New("job duplicado")
	// ErrConflict: a versão gravada não é a esperada; alguém escreveu
	// antes (lock otimista).
	ErrConflict = errors.New("versão desatualizada")
	// ErrCercado: a partição tem outro dono; o fencing token que este
	// nó carrega é antigo, e a transação foi desfeita.
	ErrCercado = errors.New(
		"partição com outro dono (fencing token antigo)")
	// ErrSemRecursos: o armazenamento está sem disco, memória ou
	// conexões; a mesma operação pode dar certo daqui a pouco.
	ErrSemRecursos = errors.New("armazenamento sem recursos")
)
