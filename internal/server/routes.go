package server

import (
	"net/http"
)

// createServeMux initializes and configures the HTTP router with all application routes.
// Returns a configured *http.ServeMux with handlers for static files, API endpoints,
// web pages, and development tools. Routes are organized by functionality for clarity.
func createServeMux() *http.ServeMux {
	router := http.NewServeMux()

	// Static assets: CSS, JS, images, and favicon
	router.HandleFunc("/static/", onlyMethod(StaticFiles, "GET", "HEAD"))
	router.HandleFunc("/favicon.ico", onlyMethod(StaticFiles, "GET", "HEAD"))

	// Health check endpoint for monitoring
	router.HandleFunc("/health", onlyMethod(Health, "GET"))

	// API endpoints
	router.HandleFunc("/api/suggestions", onlyMethod(SuggestionsAPI, "GET"))

	// Search endpoints (supports both GET and POST)
	router.HandleFunc("/search", onlyMethod(Search, "GET", "POST"))

	// Development tools (only active in dev mode)
	router.HandleFunc("/dev", onlyMethod(DevIndex, "GET"))
	router.HandleFunc("/dev/panic", DevPanic) // No method guard - allows any method for testing
	router.HandleFunc("/dev/404", Dev404)
	router.HandleFunc("/dev/500", Dev500)
	router.HandleFunc("/dev/tmpl-error", Dev500Tmpl)

	// Main application pages with filter support
	router.HandleFunc("/artists", onlyMethod(Artists, "GET", "POST"))
	router.HandleFunc("/artists/", onlyMethod(ArtistDetail, "GET"))
	router.HandleFunc("/locations", onlyMethod(Locations, "GET", "POST"))
	router.HandleFunc("/locations/", onlyMethod(LocationDetail, "GET"))

	// Home page (catch-all root handler)
	router.HandleFunc("/", onlyMethod(Home, "GET"))

	return router
}
