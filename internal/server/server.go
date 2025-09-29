package server

import (
	"context"
	"fmt"
	"groupie-tracker/internal/config"
	"groupie-tracker/internal/data"
	"html/template"
	"log"
	"net/http"
	"time"
)

// Server encapsulates all server dependencies with direct repository access
// and cached expensive computations for optimal performance.
type Server struct {
	// Direct repository access (eliminates service layer facade)
	repo *data.Repository

	// Pre-compiled templates for rendering
	templates map[string]*template.Template

	// Cached expensive computations (computed once at startup)
	suggestions        []data.SearchSuggestion    // All search suggestions cached
	artistFilterOpts   data.ArtistFilterOptions   // Artist filter options cached
	locationFilterOpts data.LocationFilterOptions // Location filter options cached

	// HTTP server instance
	httpServer *http.Server
	// Handler is the http.Handler used by the server. It is exported to allow
	// external packages (tests) to create test servers using the same handler
	// without needing to start a full network listener.
	Handler http.Handler
}

// NewServer creates and fully initializes a Server with dependency injection.
//
// This function performs the complete server bootstrap process:
//   - Initializes the data repository and loads all API data
//   - Compiles all HTML templates with custom helper functions
//   - Configures HTTP timeouts and middleware chain
//   - Logs startup performance and cache statistics
//
// The server follows dependency injection pattern where dependencies
// are explicitly managed within the Server struct.
//
// Returns a configured *Server ready to call ListenAndServe(), or an error
// if data loading or template compilation fails.
func NewServer() (*Server, error) {
	start := time.Now()

	// Create server instance
	server := &Server{}

	// Initialize repository - reads config internally, no parameters needed
	server.repo = data.NewRepository()

	// Load all data from external API with timeout protection
	log.Println("Loading initial data...")
	loadCtx, loadCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer loadCancel()

	err := server.repo.LoadData(loadCtx)
	if err != nil {
		return nil, fmt.Errorf("failed to load data: %w", err)
	}

	// Initialize expensive computations cache
	server.initializeCaches()

	// Compile all HTML templates once at startup
	server.loadTemplates()

	// Log startup summary with cache status and performance metrics
	stats := server.repo.GetStats()
	if !server.repo.IsCacheEnabled() {
		log.Printf("Data loaded successfully - %d artists (Image caching is disabled, serving from API)", stats["total_artists"])
	} else {
		log.Printf("Data loaded successfully with cache - %d artists", stats["total_artists"])
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

// initializeCaches pre-computes expensive operations and stores them for O(1) access
func (s *Server) initializeCaches() {
	// Cache all search suggestions (expensive to generate on each request)
	s.suggestions = s.repo.GenerateAllSearchSuggestions()

	// Cache filter options (expensive to compute min/max ranges)
	s.artistFilterOpts = s.repo.GetArtistFilterOptions()
	s.locationFilterOpts = s.repo.GetLocationFilterOptions()
}
