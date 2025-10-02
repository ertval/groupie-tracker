package web

import (
	"context"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"sync"
	"time"

	"groupie-tracker/internal/api"
	"groupie-tracker/internal/conf"
	"groupie-tracker/internal/data"
)

// App encapsulates the complete HTTP server with its data store, compiled templates, and HTTP server instance.
// All dependencies are injected at initialization, and the app becomes fully operational after NewApp returns.
type App struct {
	store   *data.Store  // Data store with preloaded artists, locations, indexes, and search suggestions (immutable after Load)
	storeMu sync.RWMutex // Protects store during refresh operations

	templates map[string]*template.Template // Pre-compiled HTML templates mapped by name (e.g., "home.tmpl", "artists.tmpl")

	httpServer *http.Server // Production HTTP server with configured timeouts and handler chain

	// Handler exposes the complete middleware + routing chain for testing purposes.
	// Tests can create httptest.Server using this handler without starting a real network listener.
	Handler http.Handler

	// Data refresh components
	apiClient *api.Client   // API client for refreshing data
	ticker    *time.Ticker  // Triggers hourly refresh
	stopChan  chan struct{} // Signals shutdown to background goroutines
}

// NewApp creates and fully initializes the application with all dependencies injected.
// Initialization pipeline:
// 1. Load all data from external API (concurrent fetching of artists and relations)
// 2. Process data into domain models with computed fields (concerts, slugs, years, etc.)
// 3. Build indexes and metadata (by ID, by slug, search suggestions, filter options)
// 4. Cache images using adaptive worker pool (scales with CPU cores) - always enabled
// 5. Compile all HTML templates with custom template functions
// 6. Assemble middleware chain and route handlers
// Returns fully initialized App ready to serve requests, or error if initialization fails.
func NewApp(apiClient *api.Client) (*App, error) {
	start := time.Now() // Track initialization time for performance monitoring

	app := &App{}

	// Load all data from API with 10-second timeout to prevent hanging during startup
	log.Println("Loading initial data...")
	loadCtx, loadCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer loadCancel()

	store := data.NewStore(apiClient)
	if err := store.Load(loadCtx); err != nil { // Load blocks until all data is fetched and processed
		return nil, fmt.Errorf("failed to load data: %w", err)
	}
	app.store = store
	app.apiClient = apiClient

	// Compile all HTML templates once at startup for performance (avoids parsing on every request)
	app.loadTemplates()

	// Log startup summary with data statistics
	stats := app.store.Stats()
	log.Printf("Data loaded - %d artists (cached: %d images, downloaded: %d images)",
		stats.TotalArtists, stats.CachedImages, stats.DownloadedImages)

	// Start background data refresh
	app.startDataRefresh()

	// Assemble complete middleware chain and route handlers into a single http.Handler
	serveMux := withMiddleware(app.createServeMux())

	// Expose handler for testing (httptest.NewServer can use this directly)
	app.Handler = serveMux
	port := getPort() // Get port from config (default :8082)

	// Create production HTTP server with proper timeouts to prevent resource exhaustion
	app.httpServer = &http.Server{
		Addr:         port,
		Handler:      serveMux,
		ReadTimeout:  conf.ReadTimeout,  // Protects against slow client attacks
		WriteTimeout: conf.WriteTimeout, // Prevents hanging on slow responses
		IdleTimeout:  conf.IdleTimeout,  // Closes idle keep-alive connections
	}

	log.Printf("🚀 Server Initialized in %v and Ready to Open - %s", time.Since(start), "http://localhost"+port)

	return app, nil
}

// StartApp starts the HTTP server and blocks until the server stops or encounters a fatal error.
// This is a blocking operation - it will run until interrupted (Ctrl+C) or an error occurs.
func (s *App) StartApp() error {
	return s.httpServer.ListenAndServe()
}

// getStore returns the current store with read lock for thread-safe access.
// All handlers should use this method instead of accessing s.store directly.
func (s *App) getStore() *data.Store {
	s.storeMu.RLock()
	defer s.storeMu.RUnlock()
	return s.store
}

// startDataRefresh initializes and starts the background data refresh ticker.
// Refreshes data every hour (configurable via conf.DataRefreshInterval).
func (s *App) startDataRefresh() {
	s.ticker = time.NewTicker(conf.DataRefreshInterval)
	s.stopChan = make(chan struct{})

	go func() {
		for {
			select {
			case <-s.ticker.C:
				s.refreshData()
			case <-s.stopChan:
				return
			}
		}
	}()

	log.Printf("Data refresh scheduled every %v", conf.DataRefreshInterval)
}

// refreshData performs the actual data refresh by creating a new store and loading fresh data.
// On success, atomically swaps the old store with the new one.
// On failure, keeps serving the old data and logs the error.
func (s *App) refreshData() {
	log.Println("Starting scheduled data refresh...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Create new store and load fresh data
	newStore := data.NewStore(s.apiClient)
	if err := newStore.Load(ctx); err != nil {
		log.Printf("⚠️  Data refresh failed: %v (keeping old data)", err)
		return
	}

	// Atomically swap stores
	s.storeMu.Lock()
	s.store = newStore
	s.storeMu.Unlock()

	stats := newStore.Stats()
	log.Printf("✅ Data refresh complete - %d artists (cached: %d images, downloaded: %d images)",
		stats.TotalArtists, stats.CachedImages, stats.DownloadedImages)
}

// Shutdown gracefully shuts down the server and stops background refresh.
func (s *App) Shutdown(ctx context.Context) error {
	log.Println("Shutting down server...")

	// Stop refresh ticker and goroutine
	if s.ticker != nil {
		s.ticker.Stop()
	}
	if s.stopChan != nil {
		close(s.stopChan)
	}

	// Shutdown HTTP server
	return s.httpServer.Shutdown(ctx)
}
