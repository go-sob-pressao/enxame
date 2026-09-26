package grpc

import (
	"context"
	"errors"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/go-sob-pressao/enxame/internal/core/job"
	"github.com/go-sob-pressao/enxame/internal/store"
)

// livro:inicio erros

// Um erro não atravessa a rede: chega ao outro lado como um código e um
// texto. paraStatus escolhe o código do lado do servidor; deStatus, do
// lado do cliente, devolve o erro de domínio correspondente, para que
// errors.Is continue funcionando depois da rede.
var mapa = []struct {
	erro   error
	codigo codes.Code
}{
	{store.ErrNotFound, codes.NotFound},
	{job.ErrInvalidTransition, codes.FailedPrecondition},
	{store.ErrConflict, codes.Aborted},
	{store.ErrDuplicate, codes.AlreadyExists},
	{context.DeadlineExceeded, codes.DeadlineExceeded},
	{context.Canceled, codes.Canceled},
}

func paraStatus(err error) error {
	if err == nil {
		return nil
	}
	if _, ok := status.FromError(err); ok {
		return err // já é um status
	}
	for _, m := range mapa {
		if errors.Is(err, m.erro) {
			return status.Error(m.codigo, err.Error())
		}
	}
	return status.Error(codes.Internal, err.Error())
}

// DeStatus traduz o status recebido de volta para o erro de domínio.
func DeStatus(err error) error {
	s, ok := status.FromError(err)
	if !ok || err == nil {
		return err
	}
	for _, m := range mapa {
		if s.Code() == m.codigo {
			return errors.Join(m.erro, err)
		}
	}
	return err
}

// livro:fim erros
