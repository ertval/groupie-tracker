// Package main is the entry point for the Groupie Tracker server application.
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
	defaultPort       = ":8080"
	defaultAPIURL     = "https://groupietrackers.herokuapp.com"
	requestTimeout    = 30 * time.Second
	shutdownTimeout   = 10 * time.Second
)

func main() {
	log.Println("Starting Groupie Tracker server...")

	// Initialize store
	store := storage.NewStore()

	// Initialize API client
	apiClient := api.NewClient(defaultAPIURL, requestTimeout)

	// Load data from API
	log.Println("Loading data from API...")
	if err := loadDataFromAPI(store, apiClient); err != nil {
		log.Fatalf("Failed to load data from API: %v", err)
	}
	log.Println("Data loaded successfully")

	// Initialize handlers
	h := handlers.NewHandlers(store)

	// Create router
	mux := createRouter(h)

	// Create server
	port := getPort()
	server := &http.Server{
		Addr:         port,
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in a goroutine
	go func() {
		log.Printf("Server starting on port %s", port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed to start: %v", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Server is shutting down...")

	// Create a deadline to wait for
	ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	// Attempt graceful shutdown
	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited")
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
	ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
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
		return defaultPort
	}
	
	// Add colon if not present
	if !strings.HasPrefix(port, ":") {
		port = ":" + port
	}
	
	return port
}
