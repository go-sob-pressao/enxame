// Package auditoria é a alternativa rejeitada pela ADR-001: estado
// mutável, com um log de auditoria escrito ao lado.
package auditoria

// livro:inicio auditoria

// Job guarda o estado corrente; Auditoria, o que aconteceu com ele.
// Os dois são escritos pelo mesmo código, e nada obriga um a
// acompanhar o outro.
type Job struct {
	Estado    string
	Tentativa int
	Auditoria []string
}

// Iniciar começa uma tentativa, e registra.
func (j *Job) Iniciar() {
	j.Estado, j.Tentativa = "running", j.Tentativa+1
	j.Auditoria = append(j.Auditoria, "tentativa iniciada")
}

// Resgatar devolve à fila o job de um worker morto. Escrito meses
// depois, por outra pessoa: muda o estado e não registra nada.
func (j *Job) Resgatar() {
	j.Estado = "available"
}

// livro:fim auditoria
