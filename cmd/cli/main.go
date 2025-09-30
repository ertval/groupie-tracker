// Package main is the entry point for the Groupie Tracker server application.
// This refactored version uses dependency injection and proper separation of concerns.
package main

import (
	"context"
	"log"
	"time"

	"groupie-tracker/internal/api"
	"groupie-tracker/internal/http"
	"groupie-tracker/internal/service"
	"groupie-tracker/internal/store"
)

// App contains all application dependencies and provides the main entry point.
type App struct {
	store         *store.DataStore
	apiClient     *api.Client
	dataService   *service.DataService
	searchService *service.SearchService
	filterService *service.FilterService
	templates     *http.TemplateRenderer
	handler       *http.Handler
	server        *http.Server
}

// NewApp creates a new application with all dependencies properly initialized.
func NewApp() (*App, error) {
	app := &App{}

	// Initialize core services
	app.store = store.New()
	app.apiClient = api.NewClient("https://groupietrackers.herokuapp.com/api", 10*time.Second)
	app.dataService = service.NewDataService()
	app.searchService = service.NewSearchService()
	app.filterService = service.NewFilterService()

	// Initialize HTTP layer
	var err error
	app.templates, err = http.NewTemplateRenderer()
	if err != nil {
		return nil, err
	}

	app.handler = http.NewHandler(app.store, app.templates, app.searchService, app.filterService)
	app.server = http.NewServer(app.handler)

	return app, nil
}

// LoadData loads all data from the API and initializes the application.
func (app *App) LoadData(ctx context.Context) error {
	log.Println("Loading data from API...")
	start := time.Now()

	// Fetch data from external API
	apiArtists, apiRelations, err := app.apiClient.FetchAllData(ctx)
	if err != nil {
		return err
	}

	log.Printf("Fetched %d artists and %d relations", len(apiArtists), len(apiRelations))

	// Process API data into domain models
	log.Println("Processing data...")
	artists, locations, stats := app.dataService.ProcessAPIData(apiArtists, apiRelations)

	// Load processed data into store
	app.store.LoadData(artists, locations, stats)

	// Generate search suggestions
	log.Println("Generating search suggestions...")
	suggestions := app.searchService.GenerateSuggestions(artists)
	app.store.LoadSuggestions(suggestions)

	log.Printf("Data loading completed in %v", time.Since(start))
	log.Printf("Loaded: %d artists, %d locations, %d suggestions",
		len(artists), len(locations), len(suggestions))

	return nil
}

// Start starts the HTTP server.
func (app *App) Start() error {
	return app.server.Start()
}

// Shutdown gracefully shuts down the application.
func (app *App) Shutdown(ctx context.Context) error {
	return app.server.Shutdown(ctx)
}

func main() {
	// Create application with dependency injection
	app, err := NewApp()
	if err != nil {
		log.Fatalf("Failed to create application: %v", err)
	}

	// Load data with timeout
	loadCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := app.LoadData(loadCtx); err != nil {
		log.Fatalf("Failed to load data: %v", err)
	}

	// Start server
	log.Printf("Starting server on %s", app.server.GetAddr())
	if err := app.Start(); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
