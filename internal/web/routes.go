package web

import (
	"net/http"

	"groupie-tracker/internal/conf"
)

// createServeMux initializes and configures the HTTP router with all application routes.
// Returns a configured *http.ServeMux with handlers for static files, API endpoints,
// web pages, and development tools. Routes are organized by functionality for clarity.
func (a *App) createServeMux() *http.ServeMux {
	mux := http.NewServeMux()

	// Static assets: CSS, JS, images, and favicon
	mux.HandleFunc("/static/", a.getHead(a.StaticFiles))
	mux.HandleFunc("/favicon.ico", a.getHead(a.StaticFiles))

	// Health check endpoint for monitoring
	mux.HandleFunc("/health", a.get(a.Health))

	// API endpoints
	// Suggestions endpoint is rate-limited per-client to protect autocomplete from abuse
	suggestionsHandler := withRateLimit(http.HandlerFunc(a.get(a.SuggestionsAPI)), float64(conf.RateLimitRequestsPerSecond), float64(conf.RateLimitBurst))
	mux.Handle("/api/suggestions", suggestionsHandler)

	// Refresh endpoint - protect with rate limiting too (manual admin endpoint)
	refreshHandler := withRateLimit(http.HandlerFunc(a.post(a.RefreshData)), float64(conf.RateLimitRequestsPerSecond), float64(conf.RateLimitBurst))
	mux.Handle("/api/refresh", refreshHandler)

	// Search endpoints (supports both GET and POST)
	mux.HandleFunc("/search", a.getPost(a.Search))

	// Development tools (only active in dev mode)
	mux.HandleFunc("/dev", a.get(a.DevIndex))
	mux.HandleFunc("/dev/panic", a.any(a.DevPanic)) // No method guard - allows any method for testing
	mux.HandleFunc("/dev/404", a.any(a.Dev404))
	mux.HandleFunc("/dev/500", a.any(a.Dev500))
	mux.HandleFunc("/dev/tmpl-error", a.any(a.Dev500Tmpl))

	// Main application pages with filter support
	mux.HandleFunc("/artists", a.getPost(a.Artists))
	mux.HandleFunc("/artists/", a.get(a.ArtistDetail))
	mux.HandleFunc("/locations", a.getPost(a.Locations))
	mux.HandleFunc("/locations/", a.get(a.LocationDetail))

	// Home page (catch-all root handler)
	mux.HandleFunc("/", a.get(a.Home))

	return mux
}
