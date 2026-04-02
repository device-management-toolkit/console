package httpserver

import (
	"net"
	"time"

	appLogger "github.com/device-management-toolkit/console/pkg/logger"
)

// Option -.
type Option func(*Server)

// Port -.
func Port(host, port string) Option {
	return func(s *Server) {
		s.server.Addr = net.JoinHostPort(host, port)
	}
}

// TLS enables TLS and optionally sets cert and key file paths.
func TLS(enable bool, certFile, keyFile string) Option {
	return func(s *Server) {
		s.useTLS = enable
		s.certFile = certFile
		s.keyFile = keyFile
	}
}

// Listener injects a pre-bound listener (useful for tests to avoid binding real ports).
func Listener(l net.Listener) Option {
	return func(s *Server) {
		s.listener = l
	}
}

// ReadTimeout -.
func ReadTimeout(timeout time.Duration) Option {
	return func(s *Server) {
		s.server.ReadTimeout = timeout
	}
}

// WriteTimeout -.
func WriteTimeout(timeout time.Duration) Option {
	return func(s *Server) {
		s.server.WriteTimeout = timeout
	}
}

// ShutdownTimeout -.
func ShutdownTimeout(timeout time.Duration) Option {
	return func(s *Server) {
		s.shutdownTimeout = timeout
	}
}

// Logger injects a logger to be used by the HTTP server internals.
func Logger(l appLogger.Interface) Option {
	return func(s *Server) {
		s.log = l
	}
}
