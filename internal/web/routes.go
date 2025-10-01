package web

import (
	"net/http"
)

// createServeMux initializes and configures the HTTP router with all application routes.
// Returns a configured *http.ServeMux with handlers for static files, API endpoints,
// web pages, and development tools. Routes are organized by functionality for clarity.
func (s *Server) createServeMux() *http.ServeMux {
	router := http.NewServeMux()

	// Static assets: CSS, JS, images, and favicon
	router.HandleFunc("/static/", s.restrictMethod(s.StaticFiles, "GET", "HEAD"))
	router.HandleFunc("/favicon.ico", s.restrictMethod(s.StaticFiles, "GET", "HEAD"))

	// Health check endpoint for monitoring
	router.HandleFunc("/health", s.restrictMethod(s.Health, "GET"))

	// API endpoints
	router.HandleFunc("/api/suggestions", s.restrictMethod(s.SuggestionsAPI, "GET"))

	// Search endpoints (supports both GET and POST)
	router.HandleFunc("/search", s.restrictMethod(s.Search, "GET", "POST"))

	// Development tools (only active in dev mode)
	router.HandleFunc("/dev", s.restrictMethod(s.DevIndex, "GET"))
	router.HandleFunc("/dev/panic", s.DevPanic) // No method guard - allows any method for testing
	router.HandleFunc("/dev/404", s.Dev404)
	router.HandleFunc("/dev/500", s.Dev500)
	router.HandleFunc("/dev/tmpl-error", s.Dev500Tmpl)

	// Main application pages with filter support
	router.HandleFunc("/artists", s.restrictMethod(s.Artists, "GET", "POST"))
	router.HandleFunc("/artists/", s.restrictMethod(s.ArtistDetail, "GET"))
	router.HandleFunc("/locations", s.restrictMethod(s.Locations, "GET", "POST"))
	router.HandleFunc("/locations/", s.restrictMethod(s.LocationDetail, "GET"))

	// Home page (catch-all root handler)
	router.HandleFunc("/", s.restrictMethod(s.Home, "GET"))

	return router
}
