// Package main provides server configuration and setup functionality.
package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"groupie-tracker/internal/api"
	"groupie-tracker/internal/handlers"
	"groupie-tracker/internal/storage"
)

const (
	DefaultPort     = ":8080"
	DefaultAPIURL   = "https://groupietrackers.herokuapp.com"
	RequestTimeout  = 30 * time.Second
	ShutdownTimeout = 10 * time.Second
	ReadTimeout     = 15 * time.Second
	WriteTimeout    = 15 * time.Second
	IdleTimeout     = 60 * time.Second
)

// Server represents the HTTP server with all its dependencies.
type Server struct {
	store     *storage.Store
	apiClient *api.Client
	handlers  *handlers.Handlers
	server    *http.Server
}

// NewServer creates and configures a new server instance.
func NewServer() (*Server, error) {
	// Initialize store
	store := storage.NewStore()

	// Initialize API client
	apiClient := api.NewClient(DefaultAPIURL, RequestTimeout)

	// Load data from API
	log.Println("Loading data from API...")
	if err := loadDataFromAPI(store, apiClient); err != nil {
		return nil, fmt.Errorf("failed to load data from API: %w", err)
	}
	log.Println("Data loaded successfully")

	// Initialize handlers
	h := handlers.NewHandlers(store)
	h.SetAPIClient(apiClient)

	// Create router
	mux := createRouter(h)

	// Create HTTP server
	port := getPort()
	server := &http.Server{
		Addr:         port,
		Handler:      mux,
		ReadTimeout:  ReadTimeout,
		WriteTimeout: WriteTimeout,
		IdleTimeout:  IdleTimeout,
	}

	return &Server{
		store:     store,
		apiClient: apiClient,
		handlers:  h,
		server:    server,
	}, nil
}

// Start starts the server and handles graceful shutdown.
func (s *Server) Start() error {
	// Start server in a goroutine
	go func() {
		// Build a clickable URL for convenience (works in most terminals)
		addr := s.server.Addr
		url := addr
		if strings.HasPrefix(addr, ":") {
			url = "http://localhost" + addr
		} else if !strings.HasPrefix(addr, "http://") && !strings.HasPrefix(addr, "https://") {
			url = "http://" + addr
		}

		log.Printf("Server starting — open %s in your browser", url)
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed to start: %v", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Server is shutting down...")

	// Create a deadline to wait for
	ctx, cancel := context.WithTimeout(context.Background(), ShutdownTimeout)
	defer cancel()

	// Attempt graceful shutdown
	if err := s.server.Shutdown(ctx); err != nil {
		return fmt.Errorf("server forced to shutdown: %w", err)
	}

	log.Println("Server exited")
	return nil
}

// createRouter sets up all routes and middleware.
func createRouter(h *handlers.Handlers) *http.ServeMux {
	mux := http.NewServeMux()

	// Static file serving
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static/"))))

	// Web routes
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			h.NotFoundHandler(w, r)
			return
		}
		h.HomeHandler(w, r)
	})
	mux.HandleFunc("/artists", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/artists" {
			h.NotFoundHandler(w, r)
			return
		}
		h.ArtistsHandler(w, r)
	})
	mux.HandleFunc("/artists/", h.ArtistDetailHandler)
	mux.HandleFunc("/locations", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/locations" {
			h.NotFoundHandler(w, r)
			return
		}
		h.LocationsHandler(w, r)
	})

	// API routes
	mux.HandleFunc("/api/search", h.SearchHandler)
	mux.HandleFunc("/api/suggest", h.SuggestHandler)
	mux.HandleFunc("/api/refresh", h.RefreshHandler)

	// Health check
	mux.HandleFunc("/healthz", h.HealthHandler)

	// Wrap with middleware
	return wrapWithMiddleware(mux)
}

// wrapWithMiddleware wraps the entire mux with middleware.
func wrapWithMiddleware(handler http.Handler) *http.ServeMux {
	mux := http.NewServeMux()

	// Wrap all requests with middleware
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		loggingMiddleware(
			recoveryMiddleware(handler),
		).ServeHTTP(w, r)
	})

	return mux
}

// recoveryMiddleware recovers from panics and returns a 500 error.
func recoveryMiddleware(next http.Handler) http.Handler {
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

// recoveryMiddlewareWithHandler recovers from panics using custom error handler.
func recoveryMiddlewareWithHandler(next http.Handler, h *handlers.Handlers) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				log.Printf("Panic recovered: %v", err)
				h.InternalErrorHandler(w, r, fmt.Sprintf("Panic: %v", err))
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// loggingMiddleware logs HTTP requests.
func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Call the next handler
		next.ServeHTTP(w, r)

		// Log the request
		log.Printf("%s %s %v", r.Method, r.URL.Path, time.Since(start))
	})
}

// loadDataFromAPI loads all data from the API into the store.
func loadDataFromAPI(store *storage.Store, client *api.Client) error {
	ctx, cancel := context.WithTimeout(context.Background(), RequestTimeout)
	defer cancel()

	data, err := client.FetchAllData(ctx)
	if err != nil {
		return fmt.Errorf("fetching data: %w", err)
	}

	storeData := storage.StoreData{
		Artists:   data.Artists,
		Locations: data.Locations,
		Dates:     data.Dates,
		Relations: data.Relations,
	}

	store.LoadData(storeData)

	log.Printf("Loaded %d artists, %d locations, %d dates, %d relations",
		len(data.Artists), len(data.Locations), len(data.Dates), len(data.Relations))

	return nil
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
