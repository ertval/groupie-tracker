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

	"groupie-tracker/internal/api"
	"groupie-tracker/internal/data"
	"groupie-tracker/internal/handlers"
)

const (
	DefaultPort    = ":8080"
	DefaultAPIURL  = "https://groupietrackers.herokuapp.com"
	RequestTimeout = 30 * time.Second
	ReadTimeout    = 15 * time.Second
	WriteTimeout   = 15 * time.Second
	IdleTimeout    = 60 * time.Second
)

// ANSI color codes for pretty CLI output (standard library only)
const (
	colorReset = "\033[0m"
	colorGreen = "\033[32m"
	colorCyan  = "\033[36m"
	// colorYellow = "\033[33m"
)

// Server represents the HTTP server with all its dependencies.
type Server struct {
	repo      *data.Repository
	apiClient *api.Client
	handlers  *handlers.Handlers
	server    *http.Server
}

// NewServer creates and configures a new server instance.
func NewServer() (*Server, error) {
	// Initialize API client
	apiClient := api.NewClient(DefaultAPIURL, RequestTimeout)

	// Initialize repository
	repo := data.NewRepository()

	// Load initial data directly from API client
	log.Println(colorCyan + "⏳ Loading initial data..." + colorReset)
	loadCtx, loadCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer loadCancel()

	// Fetch all data from API
	apiData, err := apiClient.FetchAll(loadCtx)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch data from API: %w", err)
	}

	// Load data into repository
	err = repo.InitializeWithData(loadCtx, apiData)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize repository with data: %w", err)
	}

	// Check if data was loaded
	artists := repo.GetAllArtists()
	if len(artists) == 0 {
		return nil, fmt.Errorf("failed to load initial data from API")
	}

	log.Printf(colorCyan+"✅ Data loaded successfully - %d artists"+colorReset, len(artists))

	// Initialize handlers with just the repository
	handlers := handlers.NewHandlers(repo)

	// Create HTTP server
	port := getPort()
	server := &http.Server{
		Addr:         port,
		Handler:      createRouter(handlers),
		ReadTimeout:  ReadTimeout,
		WriteTimeout: WriteTimeout,
		IdleTimeout:  IdleTimeout,
	}

	return &Server{
		repo:      repo,
		apiClient: apiClient,
		handlers:  handlers,
		server:    server,
	}, nil
}

// Start starts the server.
func (s *Server) Start() error {
	// Build a clickable URL for convenience (works in most terminals)
	addr := s.server.Addr
	url := addr
	if strings.HasPrefix(addr, ":") {
		url = "http://localhost" + addr
	} else if !strings.HasPrefix(addr, "http://") && !strings.HasPrefix(addr, "https://") {
		url = "http://" + addr
	}

	log.Printf(colorGreen+"🚀 Server starting — open %s in your browser"+colorReset, url)

	// Start server (blocking)
	err := s.server.ListenAndServe()
	if err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("server failed to start: %w", err)
	}
	return nil
}

// createRouter sets up all routes and middleware.
func createRouter(h *handlers.Handlers) *http.ServeMux {
	mux := http.NewServeMux()

	// Static file serving
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static/"))))

	// Web routes
	mux.HandleFunc("/", h.HomeHandler)
	mux.HandleFunc("/artists", h.ArtistsHandler)
	mux.HandleFunc("/artists/", h.ArtistDetailHandler)
	mux.HandleFunc("/locations", h.LocationsHandler)
	mux.HandleFunc("/locations/", h.LocationDetailHandler)

	// Health check
	mux.HandleFunc("/healthz", h.HealthHandler)
	// Development: panic trigger endpoint (DEV ONLY)
	mux.HandleFunc("/dev/panic", h.PanicHandler)

	// Apply middleware to all routes
	return applyMiddleware(mux, h)
}

// applyMiddleware applies logging and recovery middleware to all routes.
func applyMiddleware(handler http.Handler, h *handlers.Handlers) *http.ServeMux {
	mux := http.NewServeMux()

	// Apply middleware to all requests
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Panic recovery
		defer func() {
			if err := recover(); err != nil {
				h.InternalErrorHandler(w, r, fmt.Sprintf("Panic recovered: %v", err))
			}
		}()

		// Serve request
		handler.ServeHTTP(w, r)

		// Log request
		log.Printf("%s %s %v", r.Method, r.URL.Path, time.Since(start))
	})

	return mux
}

// getPort returns the port to run the server on.
func getPort() string {
	port := os.Getenv("PORT")
	if port == "" {
		return DefaultPort
	}

	// Add colon if not present
	if !strings.HasPrefix(port, ":") {
		port = ":" + port
	}

	return port
}
