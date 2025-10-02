package web

import (
	"context"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"time"

	"groupie-tracker/internal/api"
	"groupie-tracker/internal/app"
	"groupie-tracker/internal/config"
	"groupie-tracker/internal/data"
	"groupie-tracker/internal/service"
)

// Server encapsulates server dependencies with data services and cached data.
type Server struct {
	// Data store and business service exposing read-only operations
	store *data.Store
	svc   *service.Service

	// Pre-compiled templates for rendering
	templates map[string]*template.Template

	// HTTP server instance
	httpServer *http.Server
	// Handler is the http.Handler used by the server. It is exported to allow
	// external packages (tests) to create test servers using the same handler
	// without needing to start a full network listener.
	Handler http.Handler
}

// NewServer creates and fully initializes a Server with dependency injection.
func NewServer(apiClient *api.Client, withCache bool) (*Server, error) {
	start := time.Now()

	// Create server instance
	server := &Server{}

	// Initialize service with injected API client
	log.Println("Loading initial data...")
	loadCtx, loadCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer loadCancel()

	store, svc, err := app.Initialize(loadCtx, apiClient, withCache)
	if err != nil {
		return nil, fmt.Errorf("failed to load data: %w", err)
	}
	server.store = store
	server.svc = svc

	// Compile all HTML templates once at startup
	server.loadTemplates()

	// Log startup summary with cache status and performance metrics
	stats := server.svc.Stats()
	if !server.svc.CacheEnabled() {
		log.Printf("Data loaded - %d artists (caching disabled)", stats.TotalArtists)
	} else {
		log.Printf("Data loaded with cache - %d artists", stats.TotalArtists)
	}

	// Assemble middleware chain and route handlers
	serveMux := withMiddleware(server.createServeMux())

	// Expose the handler so tests can reuse it directly (httptest.NewServer)
	server.Handler = serveMux
	port := getPort()

	// Create production-ready HTTP server with configured timeouts
	server.httpServer = &http.Server{
		Addr:         port,
		Handler:      serveMux,
		ReadTimeout:  config.ReadTimeout,
		WriteTimeout: config.WriteTimeout,
		IdleTimeout:  config.IdleTimeout,
	}

	log.Printf("🚀 Server Initialized in %v and Ready to Open - %s", time.Since(start), "http://localhost"+port)

	return server, nil
}

// ListenAndServe starts the HTTP server (blocking operation)
func (s *Server) ListenAndServe() error {
	return s.httpServer.ListenAndServe()
}
