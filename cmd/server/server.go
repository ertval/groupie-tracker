package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

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
func newServer(apiURL string) (*http.Server, error) {
	// Initialize data repository
	repo := data.NewRepository(apiURL, requestTimeout)

	// Load data from API
	log.Println("Loading initial data...")
	loadCtx, loadCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer loadCancel()

	cached, downloaded, failed, err := repo.LoadData(loadCtx)
	if err != nil {
		return nil, fmt.Errorf("failed to load data: %w", err)
	}

	// Pretty single-line startup summary
	totalArtists := len(repo.GetArtists())
	if failed == 0 {
		log.Printf("Data loaded successfully - %d artists (Images: %d cached, %d downloaded)", totalArtists, cached, downloaded)
	} else {
		log.Printf("Data loaded successfully - %d artists (Images: %d cached, %d downloaded, %d failed)", totalArtists, cached, downloaded, failed)
	}

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

	// Static file serving
	mux.HandleFunc("/static/", h.StaticFiles)

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
		return defaultPort
	}

	if !strings.HasPrefix(port, ":") {
		port = ":" + port
	}

	return port
}
