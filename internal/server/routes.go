package server

import (
	"net/http"
)

// Routes sets up all Routes.
func (a *App) Routes() *http.ServeMux {
	mux := http.NewServeMux()

	// Static file serving
	mux.HandleFunc("/static/", a.StaticFiles)
	mux.HandleFunc("/favicon.ico", a.StaticFiles)

	// Health check
	mux.HandleFunc("/health", a.Health)

	// Dev routes
	mux.HandleFunc("/dev", a.DevIndex)
	mux.HandleFunc("/dev/panic", a.DevPanic)
	mux.HandleFunc("/dev/404", a.Dev404)
	mux.HandleFunc("/dev/500", a.Dev500)
	mux.HandleFunc("/dev/tmpl-error", a.Dev500Tmpl)

	// Web routes
	mux.HandleFunc("/artists", a.Artists)
	mux.HandleFunc("/artists/", a.ArtistDetail)
	mux.HandleFunc("/locations", a.Locations)
	mux.HandleFunc("/locations/", a.LocationDetail)

	// Home route
	mux.HandleFunc("/", a.Home)

	return mux
}
