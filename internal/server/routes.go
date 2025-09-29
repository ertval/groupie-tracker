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
	router.HandleFunc("/static/", methodGuardMultiple([]string{"GET", "HEAD"}, StaticFiles))
	router.HandleFunc("/favicon.ico", methodGuardMultiple([]string{"GET", "HEAD"}, StaticFiles))

	// Health check endpoint for monitoring
	router.HandleFunc("/health", methodGuard("GET", Health))

	// API endpoints
	router.HandleFunc("/api/suggestions", methodGuard("GET", SuggestionsAPI))

	// Search endpoints (supports both GET and POST)
	router.HandleFunc("/search", methodGuardMultiple([]string{"GET", "POST"}, Search))

	// Development tools (only active in dev mode)
	router.HandleFunc("/dev", methodGuard("GET", DevIndex))
	router.HandleFunc("/dev/panic", DevPanic) // No method guard - allows any method for testing
	router.HandleFunc("/dev/404", Dev404)
	router.HandleFunc("/dev/500", Dev500)
	router.HandleFunc("/dev/tmpl-error", Dev500Tmpl)

	// Main application pages with filter support
	router.HandleFunc("/artists", methodGuardMultiple([]string{"GET", "POST"}, Artists))
	router.HandleFunc("/artists/", methodGuard("GET", ArtistDetail))
	router.HandleFunc("/locations", methodGuardMultiple([]string{"GET", "POST"}, Locations))
	router.HandleFunc("/locations/", methodGuard("GET", LocationDetail))

	// Home page (catch-all root handler)
	router.HandleFunc("/", methodGuard("GET", Home))

	return router
}
