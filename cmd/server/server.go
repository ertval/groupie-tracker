package main

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
	"groupie-tracker/internal/handlers"
)

// server configuration is now provided by the internal/config package

// newServer creates and initializes a new HTTP server.
func newServer() (*http.Server, error) {

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
	handler := handlers.NewHandler(repo)
	mux := withMiddleware(createRouter(handler))
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

	// addr := httpServer.Addr
	// url := addr
	// if strings.HasPrefix(addr, ":") {
	// 	url = "http://localhost" + addr
	// } else if !strings.HasPrefix(addr, "http://") && !strings.HasPrefix(addr, "https://") {
	// 	url = "http://" + addr
	// }

	log.Printf("🚀 Server Initialized in %v and Ready to Open - %s in your browser", time.Since(start), "http://localhost"+port)

	return httpServer, nil
}

// createRouter sets up all routes.
func createRouter(h *handlers.Handler) *http.ServeMux {
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
		return config.DefaultPort
	}

	if !strings.HasPrefix(port, ":") {
		port = ":" + port
	}

	return port
}
