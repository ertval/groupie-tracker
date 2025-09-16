// Package main provides server configuration and setup functionality.
package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"groupie-tracker/internal/client"
	"groupie-tracker/internal/data"
	"groupie-tracker/internal/handlers"
)

const (
	defaultPort    = ":8080"
	defaultAPIURL  = "https://groupietrackers.herokuapp.com"
	requestTimeout = 30 * time.Second
	readTimeout    = 15 * time.Second
	writeTimeout   = 15 * time.Second
	idleTimeout    = 60 * time.Second
)

// newServer creates and initializes a new HTTP server.
func newServer() (*http.Server, error) {
	// Initialize API client
	apiClient := client.NewClient(defaultAPIURL, requestTimeout)

	// Initialize repository with data from API
	repo := data.NewRepository()

	log.Println("Loading initial data...")
	loadCtx, loadCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer loadCancel()

	if err := repo.InitializeWithAPIClient(loadCtx, apiClient); err != nil {
		return nil, fmt.Errorf("failed to initialize repository: %w", err)
	}

	log.Printf("Data loaded successfully - %d artists", len(repo.GetAllArtists()))

	// Initialize handlers with the repository
	h := handlers.NewHandlers(repo)

	// Create HTTP server
	port := getPort()
	server := &http.Server{
		Addr:         port,
		Handler:      withMiddleware(createRouter(h)),
		ReadTimeout:  readTimeout,
		WriteTimeout: writeTimeout,
		IdleTimeout:  idleTimeout,
	}

	return server, nil
}

// start starts the HTTP server.
func start(server *http.Server) error {
	// Build a clickable URL for convenience
	addr := server.Addr
	url := addr
	if strings.HasPrefix(addr, ":") {
		url = "http://localhost" + addr
	} else if !strings.HasPrefix(addr, "http://") && !strings.HasPrefix(addr, "https://") {
		url = "http://" + addr
	}

	log.Printf("Server starting — open %s in your browser", url)

	// Start server (blocking)
	err := server.ListenAndServe()
	if err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("server failed to start: %w", err)
	}
	return nil
}

// createRouter sets up all routes.
func createRouter(h *handlers.Handlers) *http.ServeMux {
	mux := http.NewServeMux()

	// Static file serving
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static/"))))

	// Favicon: serve static/favicon.ico if present, otherwise return 204 No Content.
	// This must be registered before the "/" catch-all route
	mux.HandleFunc("/favicon.ico", func(w http.ResponseWriter, r *http.Request) {
		// Prefer a physical file under static/favicon.ico so browsers get a real icon.
		faviconPath := "static/favicon.ico"
		if _, err := os.Stat(faviconPath); err == nil {
			http.ServeFile(w, r, faviconPath)
			return
		}
		// No favicon available; return 204 to indicate "no content" and avoid extra logs.
		w.WriteHeader(http.StatusNoContent)
	})

	// Health check (register before "/" to avoid catch-all)
	mux.HandleFunc("/healthz", h.HealthHandler)
	// Development: panic trigger endpoint (DEV ONLY)
	mux.HandleFunc("/dev/panic", h.PanicHandler)

	// Web routes - specific routes first, then more general ones
	mux.HandleFunc("/artists", h.ArtistsHandler)
	mux.HandleFunc("/artists/", h.ArtistDetailHandler)
	mux.HandleFunc("/locations", h.LocationsHandler)
	mux.HandleFunc("/locations/", h.LocationDetailHandler)

	// Home route - this catches everything else, so it must be last
	mux.HandleFunc("/", h.HomeHandler)

	return mux
}

// withMiddleware applies all middleware to a handler.
func withMiddleware(next http.Handler) http.Handler {
	return withLogging(withRecovery(next))
}

// withRecovery wraps a handler with panic recovery middleware.
func withRecovery(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				log.Printf("Panic recovered: %v", err)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// withLogging wraps a handler with request logging middleware.
func withLogging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s %v", r.Method, r.URL.Path, time.Since(start))
	})
}

// getPort returns the port to run the server on.
func getPort() string {
	port := os.Getenv("PORT")
	if port == "" {
		return defaultPort
	}

	// Add colon if not present
	if !strings.HasPrefix(port, ":") {
		port = ":" + port
	}

	return port
}
