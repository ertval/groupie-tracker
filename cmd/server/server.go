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

	"groupie-tracker/internal/handlers"
	"groupie-tracker/internal/repository"
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
	// Initialize data repository
	repo := repository.NewRepository(defaultAPIURL, requestTimeout)

	// Load data from API
	log.Println("Loading initial data...")
	loadCtx, loadCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer loadCancel()

	if err := repo.LoadData(loadCtx); err != nil {
		return nil, fmt.Errorf("failed to load data: %w", err)
	}

	log.Printf("Data loaded successfully - %d artists", repo.GetStats()["total_artists"])

	// Initialize handlers
	handler := handlers.NewHandler(repo)
	mux := withMiddleware(createRouter(handler))

	// Create HTTP server
	port := getPort()
	httpServer := &http.Server{
		Addr:         port,
		Handler:      mux,
		ReadTimeout:  readTimeout,
		WriteTimeout: writeTimeout,
		IdleTimeout:  idleTimeout,
	}

	// Build a clickable URL for convenience and log it here so the server
	// startup information is available immediately after initialization.
	addr := httpServer.Addr
	url := addr
	if strings.HasPrefix(addr, ":") {
		url = "http://localhost" + addr
	} else if !strings.HasPrefix(addr, "http://") && !strings.HasPrefix(addr, "https://") {
		url = "http://" + addr
	}

	log.Printf("🚀 Server ready — open %s in your browser", url)

	return httpServer, nil
}

// createRouter sets up all routes.
func createRouter(h *handlers.Handler) *http.ServeMux {
	mux := http.NewServeMux()

	// Static file serving - unified handler for all static assets
	mux.HandleFunc("/static/", h.StaticFiles)
	mux.HandleFunc("/favicon.ico", h.StaticFiles)

	// Health check (register before "/" to avoid catch-all)
	mux.HandleFunc("/health", h.Health)

	// Development endpoints (DEV ONLY)
	mux.HandleFunc("/dev/panic", h.DevPanic)
	mux.HandleFunc("/dev/404", h.Dev404)
	mux.HandleFunc("/dev/500", h.Dev500)
	mux.HandleFunc("/dev/template-error", h.DevTemplateError)

	// Web routes - specific routes first, then more general ones
	mux.HandleFunc("/artists", h.Artists)
	mux.HandleFunc("/artists/", h.ArtistDetail)
	mux.HandleFunc("/locations", h.Locations)
	mux.HandleFunc("/locations/", h.LocationDetail)

	// Home route - this catches everything else, so it must be last
	mux.HandleFunc("/", h.Home)

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
				w.Header().Set("Content-Type", "text/plain; charset=utf-8")
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte("500 Internal Server Error"))
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
