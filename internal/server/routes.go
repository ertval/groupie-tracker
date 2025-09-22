package server

import (
	"net/http"
)

// Routes sets up all Routes.
func (a *App) Routes() *http.ServeMux {
	router := http.NewServeMux()

	// Static file serving
	router.HandleFunc("/static/", a.StaticFiles)
	router.HandleFunc("/favicon.ico", a.StaticFiles)

	// Health check
	router.HandleFunc("/health", a.Health)

	// Dev routes
	router.HandleFunc("/dev", a.DevIndex)
	router.HandleFunc("/dev/panic", a.DevPanic)
	router.HandleFunc("/dev/404", a.Dev404)
	router.HandleFunc("/dev/500", a.Dev500)
	router.HandleFunc("/dev/tmpl-error", a.Dev500Tmpl)

	// Web routes
	router.HandleFunc("/artists", a.Artists)
	router.HandleFunc("/artists/", a.ArtistDetail)
	router.HandleFunc("/locations", a.Locations)
	router.HandleFunc("/locations/", a.LocationDetail)

	// Home route
	router.HandleFunc("/", a.Home)

	return router
}
