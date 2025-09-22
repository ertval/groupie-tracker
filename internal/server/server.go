package server

import (
	"context"
	"fmt"
	"groupie-tracker/internal/config"
	"groupie-tracker/internal/data"
	"html/template"
	"log"
	"net/http"
	"time"
)

// App holds the application state and handlers.
type App struct {
	repo      *data.Repository
	templates map[string]*template.Template
}

// NewApp creates a new app with the given repository.
func NewApp(repo *data.Repository) *App {
	h := &App{repo: repo}
	h.loadTemplates()
	return h
}

// server configuration is now provided by the internal/server package

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

	// Initialize handlers with routes and middleware
	serveMux := withMiddleware(NewApp(repo).Routes())
	port := getPort()
	// log.Printf("Server is starting on port %s", port)

	// Create HTTP server using values from config
	httpServer := &http.Server{
		Addr:         port,
		Handler:      serveMux,
		ReadTimeout:  config.ReadTimeout,
		WriteTimeout: config.WriteTimeout,
		IdleTimeout:  config.IdleTimeout,
	}

	log.Printf("🚀 Server Initialized in %v and Ready to Open - %s", time.Since(start), "http://localhost"+port)

	return httpServer, nil
}
