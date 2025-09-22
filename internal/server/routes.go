package server

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"groupie-tracker/internal/config"
	"groupie-tracker/internal/data"
)

// server configuration is now provided by the internal/config package

// NewServer creates and initializes a new HTTP server.
func NewServer() (*http.Server, error) {

	start := time.Now()

	// Initialize data repository (reads config internally)
	repo := data.NewRepository()

	// Load data from API
	log.Println("Loading initial data...")
	loadCtx, loadCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer loadCancel()

	err := repo.LoadData(loadCtx)
	if err != nil {
		return nil, fmt.Errorf("failed to load data: %w", err)
	}

	// Pretty single-line startup summary
	stats := repo.GetStats()
	switch repo.CacheStatus {
	case data.CacheDisabled:
		log.Printf("Data loaded successfully - %d artists (Image caching is disabled, serving from API)", stats["total_artists"])
	case data.CacheCold:
		log.Printf("Data loaded successfully with Cold cache - %d artists (Downloaded %d images)", stats["total_artists"], stats["downloaded_images"])
	case data.CacheWarm:
		log.Printf("Data loaded successfully with Warm cache - %d artists (Loaded %d images from cache)", stats["total_artists"], stats["cached_images"])
	}

	// Initialize handlers, routes, and middleware
	handler := NewHandler(repo)
	mux := withMiddleware(setupRoutes(handler))
	port := getPort()
	// log.Printf("Server is starting on port %s", port)

	// Create HTTP server using values from config
	httpServer := &http.Server{
		Addr:         port,
		Handler:      mux,
		ReadTimeout:  config.ReadTimeout,
		WriteTimeout: config.WriteTimeout,
		IdleTimeout:  config.IdleTimeout,
	}

	log.Printf("🚀 Server Initialized in %v and Ready to Open - %s", time.Since(start), "http://localhost"+port)

	return httpServer, nil
}

// setupRoutes sets up all routes.
func setupRoutes(h *Handler) *http.ServeMux {
	mux := http.NewServeMux()

	// Static file serving
	mux.HandleFunc("/static/", h.StaticFiles)
	mux.HandleFunc("/favicon.ico", h.StaticFiles)

	// Health check
	mux.HandleFunc("/health", h.Health)

	// Dev routes
	mux.HandleFunc("/dev", h.DevIndex)
	mux.HandleFunc("/dev/panic", h.DevPanic)
	mux.HandleFunc("/dev/404", h.Dev404)
	mux.HandleFunc("/dev/500", h.Dev500)
	mux.HandleFunc("/dev/tmpl-error", h.Dev500Tmpl)

	// Web routes
	mux.HandleFunc("/artists", h.Artists)
	mux.HandleFunc("/artists/", h.ArtistDetail)
	mux.HandleFunc("/locations", h.Locations)
	mux.HandleFunc("/locations/", h.LocationDetail)

	// Home route
	mux.HandleFunc("/", h.Home)

	return mux
}

// getPort returns the port to run the server on.
func getPort() string {
	port := os.Getenv("PORT")
	if port == "" {
		return config.DefaultPort
	}

	if !strings.HasPrefix(port, ":") {
		port = ":" + port
	}

	return port
}
