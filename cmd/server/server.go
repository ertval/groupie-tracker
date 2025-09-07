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

// ANSI color codes for pretty CLI output (standard library only)
const (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorCyan   = "\033[36m"
)

// apiClientAdapter adapts the api.Client to the storage.APIClient interface
type apiClientAdapter struct {
	client *api.Client
}

func (a *apiClientAdapter) FetchAllData(ctx context.Context) (*storage.APIData, error) {
	data, err := a.client.FetchAllData(ctx)
	if err != nil {
		return nil, err
	}

	// Convert api.APIData to storage.APIData
	return &storage.APIData{
		Artists:   data.Artists,
		Locations: data.Locations,
		Dates:     data.Dates,
		Relations: data.Relations,
	}, nil
}

// Server represents the HTTP server with all its dependencies.
type Server struct {
	store     *storage.Store
	apiClient *api.Client
	handlers  *handlers.Handlers
	server    *http.Server
	ctx       context.Context
	cancel    context.CancelFunc
}

// NewServer creates and configures a new server instance.
func NewServer() (*Server, error) {
	// Initialize API client
	apiClient := api.NewClient(DefaultAPIURL, RequestTimeout)

	// Create adapter for storage interface
	adapter := &apiClientAdapter{client: apiClient}

	// Initialize store with cache
	store := storage.NewStoreWithCache(adapter)

	// Create context for cache management
	ctx, cancel := context.WithCancel(context.Background())

	// Start cache for periodic updates
	store.StartCache(ctx)

	// Wait for initial data load with timeout
	log.Println(colorCyan + "⏳ Waiting for initial data load..." + colorReset)
	loadCtx, loadCancel := context.WithTimeout(ctx, 10*time.Second)
	defer loadCancel()

	// Wait for data to be loaded
	if err := waitForDataLoad(store, loadCtx); err != nil {
		cancel()
		return nil, err
	}

	// Check if data was loaded
	stats := store.GetStats()
	if stats["artists"] == 0 {
		cancel()
		return nil, fmt.Errorf("failed to load initial data from API")
	}

	log.Printf(colorCyan+"✅ Cache started successfully - loaded %d artists, %d locations, %d dates, %d relations"+colorReset,
		stats["artists"], stats["locations"], stats["dates"], stats["relations"])

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
		ctx:       ctx,
		cancel:    cancel,
	}, nil
}

// waitForDataLoad waits for the store to load initial data
func waitForDataLoad(store *storage.Store, ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for initial data load from API")
		default:
			stats := store.GetStats()
			if stats["artists"] > 0 {
				return nil
			}
			time.Sleep(50 * time.Millisecond)
		}
	}
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

		log.Printf(colorGreen+"🚀 Server starting — open %s in your browser"+colorReset, url)
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf(colorRed+"❌ Server failed to start: %v"+colorReset, err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println(colorYellow + "🛑 Server is shutting down..." + colorReset)

	// Stop the cache first
	log.Println(colorYellow + "🛑 Stopping cache..." + colorReset)
	s.store.StopCache()
	s.cancel() // Cancel the context

	// Create a deadline to wait for
	ctx, cancel := context.WithTimeout(context.Background(), ShutdownTimeout)
	defer cancel()

	// Attempt graceful shutdown
	if err := s.server.Shutdown(ctx); err != nil {
		return fmt.Errorf("server forced to shutdown: %w", err)
	}

	log.Println(colorGreen + "👋 Server exited" + colorReset)
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

	// Wrap with middleware (pass handlers so recovery can use InternalErrorHandler)
	return wrapWithMiddleware(mux, h)
}

// wrapWithMiddleware wraps the entire mux with middleware.
// It accepts handlers so the recovery middleware can render errors via InternalErrorHandler.
func wrapWithMiddleware(handler http.Handler, h *handlers.Handlers) *http.ServeMux {
	mux := http.NewServeMux()

	// Wrap all requests with middleware
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		loggingMiddleware(
			recoveryMiddlewareWithHandler(handler, h),
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
