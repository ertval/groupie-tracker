package web

import (
	"context"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"time"

	"groupie-tracker/internal/api"
	"groupie-tracker/internal/config"
	"groupie-tracker/internal/data"
)

// Server encapsulates server dependencies with data services and cached data.
type Server struct {
	// Business service exposing read-only data operations
	svc *data.Service

	// Pre-compiled templates for rendering
	templates map[string]*template.Template

	// Lightweight search query cache (for frequent searches)
	searchCache map[string][]data.Artist // Key: normalized query, Value: search results
	cacheSize   int                      // Maximum number of cached queries

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
	server.svc = data.NewService(apiClient, withCache)

	// Load all data from external API with timeout protection
	log.Println("Loading initial data...")
	loadCtx, loadCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer loadCancel()

	err := server.svc.Load(loadCtx)
	if err != nil {
		return nil, fmt.Errorf("failed to load data: %w", err)
	}

	// Initialize in-memory search cache
	server.initializeSearchCache()

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

// initializeSearchCache prepares the in-memory search cache.
func (s *Server) initializeSearchCache() {
	s.searchCache = make(map[string][]data.Artist)
	s.cacheSize = 50 // Reasonable cache size for frequent searches
}

// getCachedSearchResults retrieves cached search results.
func (s *Server) getCachedSearchResults(normalizedQuery string) ([]data.Artist, bool) {
	if results, found := s.searchCache[normalizedQuery]; found {
		return results, true
	}
	return nil, false
}

// setCachedSearchResults stores search results in cache.
func (s *Server) setCachedSearchResults(normalizedQuery string, results []data.Artist) {
	// Simple cache eviction: if at capacity, clear cache (could be more sophisticated)
	if len(s.searchCache) >= s.cacheSize {
		// Clear half the cache to make room (simple eviction strategy)
		newCache := make(map[string][]data.Artist, s.cacheSize)
		count := 0
		for key, value := range s.searchCache {
			if count >= s.cacheSize/2 {
				break
			}
			newCache[key] = value
			count++
		}
		s.searchCache = newCache
	}

	s.searchCache[normalizedQuery] = results
}
