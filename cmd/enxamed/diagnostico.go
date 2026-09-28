package main

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"net/http/pprof"
	"time"
)

// livro:inicio diagnostico

// diagnosticar serve os perfis do pprof num endereço próprio, que não é
// o da API: os perfis mostram o que o processo está fazendo, e não
// devem ficar ao alcance de quem fala com a API. O padrão é desligado;
// quando ligado, o endereço deve ser local (127.0.0.1:6060).
func diagnosticar(ctx context.Context, lis net.Listener,
	log *slog.Logger) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/debug/pprof/", pprof.Index)
	mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
	mux.HandleFunc("/debug/pprof/trace", pprof.Trace)
	mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	srv := &http.Server{Handler: mux,
		ReadHeaderTimeout: 5 * time.Second}
	go func() {
		<-ctx.Done()
		_ = srv.Close()
	}()
	log.InfoContext(ctx, "diagnóstico no ar",
		slog.String("endereco", lis.Addr().String()))
	if err := srv.Serve(lis); !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

// livro:fim diagnostico
