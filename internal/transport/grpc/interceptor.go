package grpc

import (
	"context"
	"crypto/subtle"
	"log/slog"
	"strings"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// livro:inicio interceptor

// Servidor devolve os interceptors do motor: autenticação por token,
// tradução de erros e log de cada chamada, com o prazo que chegou.
func Servidor(token string, log *slog.Logger) []grpc.ServerOption {
	unario := func(ctx context.Context, req any,
		info *grpc.UnaryServerInfo, h grpc.UnaryHandler) (any, error) {
		if err := autenticar(ctx, token); err != nil {
			return nil, err
		}
		inicio := time.Now()
		resp, err := h(ctx, req)
		prazo, _ := ctx.Deadline()
		log.DebugContext(ctx, "grpc",
			slog.String("metodo", info.FullMethod),
			slog.Duration("duracao", time.Since(inicio)),
			slog.Duration("prazo", time.Until(prazo)),
			slog.Any("erro", err))
		return resp, paraStatus(err)
	}
	stream := func(srv any, ss grpc.ServerStream,
		_ *grpc.StreamServerInfo, h grpc.StreamHandler) error {
		if err := autenticar(ss.Context(), token); err != nil {
			return err
		}
		return paraStatus(h(srv, ss))
	}
	return []grpc.ServerOption{grpc.ChainUnaryInterceptor(unario),
		grpc.ChainStreamInterceptor(stream)}
}

// Cliente devolve os interceptors do worker: o token em toda chamada, e
// um prazo padrão para a chamada unária que chegar sem prazo nenhum —
// sem prazo, uma rede parada seguraria o worker para sempre.
func Cliente(token string, padrao time.Duration) []grpc.DialOption {
	com := func(ctx context.Context) context.Context {
		return metadata.AppendToOutgoingContext(ctx,
			"authorization", "Bearer "+token)
	}
	unario := func(ctx context.Context, metodo string, req, resp any,
		cc *grpc.ClientConn, inv grpc.UnaryInvoker,
		opts ...grpc.CallOption) error {
		if _, ok := ctx.Deadline(); !ok {
			var cancel context.CancelFunc
			ctx, cancel = context.WithTimeout(ctx, padrao)
			defer cancel()
		}
		return DeStatus(inv(com(ctx), metodo, req, resp, cc, opts...))
	}
	stream := func(ctx context.Context, desc *grpc.StreamDesc,
		cc *grpc.ClientConn, metodo string, s grpc.Streamer,
		opts ...grpc.CallOption) (grpc.ClientStream, error) {
		return s(com(ctx), desc, cc, metodo, opts...)
	}
	return []grpc.DialOption{grpc.WithChainUnaryInterceptor(unario),
		grpc.WithChainStreamInterceptor(stream)}
}

// livro:fim interceptor

func autenticar(ctx context.Context, token string) error {
	md, _ := metadata.FromIncomingContext(ctx)
	for _, v := range md.Get("authorization") {
		recebido := strings.TrimPrefix(v, "Bearer ")
		if subtle.ConstantTimeCompare([]byte(recebido),
			[]byte(token)) == 1 {
			return nil
		}
	}
	return status.Error(codes.Unauthenticated,
		"token ausente ou inválido")
}
