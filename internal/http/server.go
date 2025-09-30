package http

import (
	"context"
	"log"
	"net/http"
	"os"
	"time"
)

const (
	// Server configuration constants
	defaultPort    = ":8082"
	readTimeout    = 10 * time.Second
	writeTimeout   = 10 * time.Second
	idleTimeout    = 60 * time.Second
	maxRequestSize = 32 << 20 // 32 MB
)

// Server wraps http.Server with additional configuration.
type Server struct {
	*http.Server
	handler *Handler
}

// NewServer creates a new HTTP server with the provided handler.
func NewServer(handler *Handler) *Server {
	// Get port from environment or use default
	port := os.Getenv("PORT")
	if port == "" {
		port = defaultPort
	}

	if !startsWith(port, ":") {
		port = ":" + port
	}

	// Create HTTP server with timeouts and size limits
	httpServer := &http.Server{
		Addr:           port,
		Handler:        handler.Routes(),
		ReadTimeout:    readTimeout,
		WriteTimeout:   writeTimeout,
		IdleTimeout:    idleTimeout,
		MaxHeaderBytes: maxRequestSize,
	}

	return &Server{
		Server:  httpServer,
		handler: handler,
	}
}

// Start starts the HTTP server.
func (s *Server) Start() error {
	log.Printf("Server starting on %s", s.Addr)
	return s.ListenAndServe()
}

// Shutdown gracefully shuts down the server.
func (s *Server) Shutdown(ctx context.Context) error {
	log.Println("Server shutting down...")
	return s.Server.Shutdown(ctx)
}

// GetAddr returns the server address.
func (s *Server) GetAddr() string {
	return s.Addr
}

// startsWith checks if a string starts with a prefix.
func startsWith(s, prefix string) bool {
	return len(s) >= len(prefix) && s[:len(prefix)] == prefix
}
