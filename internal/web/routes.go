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
	router.HandleFunc("/static/", app.getHead(app.StaticFiles))
	router.HandleFunc("/favicon.ico", app.getHead(app.StaticFiles))

	// Health check endpoint for monitoring
	router.HandleFunc("/health", app.get(app.Health))

	// API endpoints
	router.HandleFunc("/api/suggestions", app.get(app.SuggestionsAPI))

	// Search endpoints (supports both GET and POST)
	router.HandleFunc("/search", app.getPost(app.Search))

	// Development tools (only active in dev mode)
	router.HandleFunc("/dev", app.get(app.DevIndex))
	router.HandleFunc("/dev/panic", app.any(app.DevPanic)) // No method guard - allows any method for testing
	router.HandleFunc("/dev/404", app.any(app.Dev404))
	router.HandleFunc("/dev/500", app.any(app.Dev500))
	router.HandleFunc("/dev/tmpl-error", app.any(app.Dev500Tmpl))

	// Main application pages with filter support
	router.HandleFunc("/artists", app.getPost(app.Artists))
	router.HandleFunc("/artists/", app.get(app.ArtistDetail))
	router.HandleFunc("/locations", app.getPost(app.Locations))
	router.HandleFunc("/locations/", app.get(app.LocationDetail))

	// Home page (catch-all root handler)
	router.HandleFunc("/", app.get(app.Home))

	return router
}
