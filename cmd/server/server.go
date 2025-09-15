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
	"groupie-tracker/internal/service"
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

// Server represents the HTTP server with all its dependencies.
type Server struct {
	store     *storage.Store
	service   *service.Service
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

	// Initialize store with cache functionality
	store := storage.NewStoreWithCache(apiClient)

	// Load initial data using API client
	log.Println(colorCyan + "⏳ Loading initial data..." + colorReset)
	loadCtx, loadCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer loadCancel()

	data, err := apiClient.FetchAllData(loadCtx)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch initial data: %w", err)
	}

	// Load data into store
	store.LoadData(*data)

	// Check if data was loaded
	artists := store.GetAllArtists()
	if len(artists) == 0 {
		return nil, fmt.Errorf("failed to load initial data from API")
	}

	log.Printf(colorCyan+"✅ Data loaded successfully - %d artists"+colorReset, len(artists))

	// Start auto-refresh for the store
	store.StartAutoRefresh()

	// Initialize service
	service := service.NewService(store)

	// Initialize handlers
	h := handlers.NewHandlers(store, service, apiClient)

	// Create context for future extensions
	ctx, cancel := context.WithCancel(context.Background())

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
		service:   service,
		apiClient: apiClient,
		handlers:  h,
		server:    server,
		ctx:       ctx,
		cancel:    cancel,
	}, nil
}

// waitForDataLoad waits for the store to load initial data
func waitForDataLoad(service *service.Service, ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for initial data load from API")
		default:
			stats := service.GetStats()
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
	log.Println(colorYellow + "🛑 Shutting down..." + colorReset)

	// Stop auto-refresh
	s.store.StopAutoRefresh()

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
	mux.HandleFunc("/locations/", h.LocationDetailHandler)

	// Health check
	mux.HandleFunc("/healthz", h.HealthHandler)

	// Development: panic trigger endpoint (DEV ONLY)
	// This intentionally panics so the recovery middleware and InternalErrorHandler can be exercised.
	mux.HandleFunc("/dev/trigger-panic", func(w http.ResponseWriter, r *http.Request) {
		h.PanicHandler(w, r)
	})

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
