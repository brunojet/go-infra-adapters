package net_http

import (
	"context"
	"net/http"
)

type MiddlewareFunc func(http.Handler) http.Handler

// Server defines the interface for HTTP server operations
// Minimal abstraction to decouple from concrete http.Server implementation
type Server interface {
	ListenAndServe() error
	ListenAndServeTLS(certFile, keyFile string) error
	Shutdown(ctx context.Context) error
}

// netHttpServer embeds http.Server to allow optional method overrides (logging, etc)
// without breaking the public *http.Server API
type netHttpServer struct {
	http.Server
}

func NewNetHttpServer(opts ...ServerOption) Server {
	cfg := newServerConfig(opts...)

	return &netHttpServer{
		Server: http.Server{
			Addr:         cfg.addr,
			Handler:      cfg.buildFinalHandler(),
			ReadTimeout:  cfg.readTimeout,
			WriteTimeout: cfg.writeTimeout,
			IdleTimeout:  cfg.idleTimeout,
		},
	}
}
