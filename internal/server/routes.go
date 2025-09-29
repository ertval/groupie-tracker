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
	router.HandleFunc("/static/", StaticFiles)
	router.HandleFunc("/favicon.ico", StaticFiles)

	// Health check endpoint for monitoring
	router.HandleFunc("/health", Health)

	// Search endpoints
	router.HandleFunc("/search", Search)

	// Development tools (only active in dev mode)
	router.HandleFunc("/dev", DevIndex)
	router.HandleFunc("/dev/panic", DevPanic)
	router.HandleFunc("/dev/404", Dev404)
	router.HandleFunc("/dev/500", Dev500)
	router.HandleFunc("/dev/tmpl-error", Dev500Tmpl)

	// Main application pages with filter support
	router.HandleFunc("/artists", Artists)
	router.HandleFunc("/artists/", ArtistDetail)
	router.HandleFunc("/locations", Locations)
	router.HandleFunc("/locations/", LocationDetail)

	// Home page (catch-all root handler)
	router.HandleFunc("/", Home)

	return router
}
