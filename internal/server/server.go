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

// Global server state following the package-level pattern.
// These variables are initialized once during server startup and
// accessed by all handler functions throughout the application lifecycle.
var (
	repo      *data.Repository              // Data layer with thread-safe read operations
	templates map[string]*template.Template // Pre-compiled HTML templates for rendering
)

// NewServer creates and fully initializes an HTTP server ready for production use.
//
// This function performs the complete server bootstrap process:
//   - Initializes the data repository and loads all API data
//   - Compiles all HTML templates with custom helper functions
//   - Configures HTTP timeouts and middleware chain
//   - Logs startup performance and cache statistics
//
// The server follows a global state pattern where the repository and templates
// are package-level variables accessed by all handler functions.
//
// Returns a configured *http.Server ready to call ListenAndServe(), or an error
// if data loading or template compilation fails.
func NewServer() (*http.Server, error) {

	start := time.Now()

	// Initialize repository - reads config internally, no parameters needed
	repo = data.NewRepository()

	// Load all data from external API with timeout protection
	log.Println("Loading initial data...")
	loadCtx, loadCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer loadCancel()

	err := repo.LoadData(loadCtx)
	if err != nil {
		return nil, fmt.Errorf("failed to load data: %w", err)
	}

	// Compile all HTML templates once at startup
	loadTemplates()

	// Log startup summary with cache status and performance metrics
	stats := repo.GetStats()
	switch repo.CacheStatus {
	case data.CacheDisabled:
		log.Printf("Data loaded successfully - %d artists (Image caching is disabled, serving from API)", stats["total_artists"])
	case data.CacheCold:
		log.Printf("Data loaded successfully with Cold cache - %d artists (Downloaded %d images)", stats["total_artists"], stats["downloaded_images"])
	case data.CacheWarm:
		log.Printf("Data loaded successfully with Warm cache - %d artists (Loaded %d images from cache)", stats["total_artists"], stats["cached_images"])
	}

	// Assemble middleware chain and route handlers
	serveMux := withMiddleware(createServeMux())
	port := getPort()
	// log.Printf("Server is starting on port %s", port)

	// Create production-ready HTTP server with configured timeouts
	httpServer := &http.Server{
		Addr:         port,
		Handler:      serveMux,
		ReadTimeout:  config.ReadTimeout,
		WriteTimeout: config.WriteTimeout,
		IdleTimeout:  config.IdleTimeout,
	}

	log.Printf("🚀 Server Initialized in %v and Ready to Open - %s", time.Since(start), "http://localhost"+port)

	return httpServer, nil
}
