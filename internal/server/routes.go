package server

import (
	"net/http"
)

// Routes sets up all Routes.
func (a *App) Routes() *http.ServeMux {
	router := http.NewServeMux()

	// Static file serving
	router.HandleFunc("GET /static/", a.StaticFiles)
	router.HandleFunc("GET /favicon.ico", a.StaticFiles)

	// Health check
	router.HandleFunc("GET /health", a.Health)

	// Dev routes
	router.HandleFunc("/dev", a.DevIndex)
	router.HandleFunc("/dev/panic", a.DevPanic)
	router.HandleFunc("/dev/404", a.Dev404)
	router.HandleFunc("/dev/500", a.Dev500)
	router.HandleFunc("/dev/tmpl-error", a.Dev500Tmpl)

	// Web routes
	router.HandleFunc("GET /artists", a.Artists)
	router.HandleFunc("GET /artists/", a.ArtistDetail)
	router.HandleFunc("GET /locations", a.Locations)
	router.HandleFunc("GET /locations/", a.LocationDetail)

	// Home route
	//router.HandleFunc("/", a.Home)
	router.HandleFunc("GET /", a.Home)

	return router
}
