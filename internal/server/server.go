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

// Server encapsulates all server dependencies using dependency injection
// and service composition following Interface Segregation Principle.
type Server struct {
	// Service dependencies - focused interfaces instead of monolithic repository
	artists   ArtistService
	search    SearchService
	locations LocationService
	stats     StatsService
	cache     CacheService

	// Internal dependencies
	repo       *data.Repository              // Still needed for initialization
	templates  map[string]*template.Template // Pre-compiled HTML templates for rendering
	httpServer *http.Server                  // HTTP server instance
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

	// Initialize services with repository dependency
	server.artists = newArtistService(server.repo)
	server.search = newSearchService(server.repo)
	server.locations = newLocationService(server.repo)
	server.stats = newStatsService(server.repo)
	server.cache = newCacheService(server.repo)

	// Compile all HTML templates once at startup
	server.loadTemplates()

	// Log startup summary with cache status and performance metrics
	stats := server.stats.GetStats()
	if !server.cache.IsCacheEnabled() {
		log.Printf("Data loaded successfully - %d artists (Image caching is disabled, serving from API)", stats["total_artists"])
	} else {
		log.Printf("Data loaded successfully with cache - %d artists", stats["total_artists"])
	}

	// Assemble middleware chain and route handlers
	serveMux := withMiddleware(server.createServeMux())
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
