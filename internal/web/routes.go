package web

import (
	"net/http"
)

// createServeMux initializes and configures the HTTP router with all application routes.
// Returns a configured *http.ServeMux with handlers for static files, API endpoints,
// web pages, and development tools. Routes are organized by functionality for clarity.
func (app *App) createServeMux() *http.ServeMux {
	router := http.NewServeMux()

	// Static assets: CSS, JS, images, and favicon
	router.HandleFunc("/static/", app.restrictMethod(app.StaticFiles, "GET", "HEAD"))
	router.HandleFunc("/favicon.ico", app.restrictMethod(app.StaticFiles, "GET", "HEAD"))

	// Health check endpoint for monitoring
	router.HandleFunc("/health", app.restrictMethod(app.Health, "GET"))

	// API endpoints
	router.HandleFunc("/api/suggestions", app.restrictMethod(app.SuggestionsAPI, "GET"))

	// Search endpoints (supports both GET and POST)
	router.HandleFunc("/search", app.restrictMethod(app.Search, "GET", "POST"))

	// Development tools (only active in dev mode)
	router.HandleFunc("/dev", app.restrictMethod(app.DevIndex, "GET"))
	router.HandleFunc("/dev/panic", app.DevPanic) // No method guard - allows any method for testing
	router.HandleFunc("/dev/404", app.Dev404)
	router.HandleFunc("/dev/500", app.Dev500)
	router.HandleFunc("/dev/tmpl-error", app.Dev500Tmpl)

	// Main application pages with filter support
	router.HandleFunc("/artists", app.restrictMethod(app.Artists, "GET", "POST"))
	router.HandleFunc("/artists/", app.restrictMethod(app.ArtistDetail, "GET"))
	router.HandleFunc("/locations", app.restrictMethod(app.Locations, "GET", "POST"))
	router.HandleFunc("/locations/", app.restrictMethod(app.LocationDetail, "GET"))

	// Home page (catch-all root handler)
	router.HandleFunc("/", app.restrictMethod(app.Home, "GET"))

	return router
}
