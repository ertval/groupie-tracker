package handlers

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"groupie-tracker/internal/data"
)

// Handler holds the application state and handlers.
type Handler struct {
	repo      *data.Repository
	templates map[string]*template.Template
}

// NewHandler creates a new handler with the given repository.
func NewHandler(repo *data.Repository) *Handler {
	h := &Handler{repo: repo}
	h.loadTemplates()
	return h
}

// Home handles the home page.
func (h *Handler) Home(w http.ResponseWriter, r *http.Request) {
	if !h.validateRequestMethodPath(w, r, "/") {
		return
	}

	artists := h.repo.GetArtists()
	stats := h.repo.GetStats()

	data := struct {
		Title          string
		ExtraCSS       string
		ExtraJS        string
		Artists        []data.Artist
		TotalMembers   int
		TotalLocations int
	}{
		Title:          "Home",
		ExtraCSS:       "home.css",
		ExtraJS:        "",
		Artists:        artists,
		TotalMembers:   stats["total_members"],
		TotalLocations: stats["total_locations"],
	}

	h.render(w, r, "home.tmpl", data)
}

// Artists handles the artists listing page.
func (h *Handler) Artists(w http.ResponseWriter, r *http.Request) {
	if !h.validateRequestMethodPath(w, r, "/artists") {
		return
	}

	artists := h.repo.GetArtists()
	data := struct {
		Title    string
		ExtraCSS string
		ExtraJS  string
		Artists  []data.Artist
	}{
		Title:    "Artists",
		ExtraCSS: "artists.css",
		ExtraJS:  "",
		Artists:  artists,
	}

	h.render(w, r, "artists.tmpl", data)
}

// ArtistDetail handles individual artist pages.
func (h *Handler) ArtistDetail(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.Error(w, r, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/artists/")
	if path == "" {
		h.Error(w, r, http.StatusNotFound, "Page not found")
		return
	}

	// Try slug first, then ID
	artist, found := h.repo.GetArtistBySlug(path)
	if !found {
		if id, err := strconv.Atoi(path); err == nil {
			artist, found = h.repo.GetArtistByID(id)
		}
		if !found {
			h.Error(w, r, http.StatusNotFound, "Artist not found")
			return
		}
	}

	// Get navigation artists
	var prevArtist, nextArtist *data.Artist
	if artist.PrevArtistID != 0 {
		if p, ok := h.repo.GetArtistByID(artist.PrevArtistID); ok {
			prevArtist = &p
		}
	}
	if artist.NextArtistID != 0 {
		if n, ok := h.repo.GetArtistByID(artist.NextArtistID); ok {
			nextArtist = &n
		}
	}

	data := struct {
		Title      string
		ExtraCSS   string
		ExtraJS    string
		Artist     data.Artist
		PrevArtist *data.Artist
		NextArtist *data.Artist
	}{
		Title:      artist.Name,
		ExtraCSS:   "artist_detail.css",
		ExtraJS:    "",
		Artist:     artist,
		PrevArtist: prevArtist,
		NextArtist: nextArtist,
	}

	h.render(w, r, "artist_detail.tmpl", data)
}

// Locations handles the locations listing page.
func (h *Handler) Locations(w http.ResponseWriter, r *http.Request) {
	if !h.validateRequestMethodPath(w, r, "/locations") {
		return
	}

	locations := h.repo.GetLocations()
	stats := h.repo.GetStats()

	data := struct {
		Title          string
		ExtraCSS       string
		ExtraJS        string
		Locations      []data.Location
		TotalCountries int
		TotalConcerts  int
	}{
		Title:          "Locations",
		ExtraCSS:       "locations.css",
		ExtraJS:        "",
		Locations:      locations,
		TotalCountries: stats["total_countries"],
		TotalConcerts:  stats["total_concerts"],
	}

	h.render(w, r, "locations.tmpl", data)
}

// LocationDetail handles individual location pages.
func (h *Handler) LocationDetail(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.Error(w, r, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	slug := strings.TrimPrefix(r.URL.Path, "/locations/")
	if slug == "" {
		h.Error(w, r, http.StatusNotFound, "Page not found")
		return
	}

	location, found := h.repo.GetLocationBySlug(slug)
	if !found {
		h.Error(w, r, http.StatusNotFound, "Location not found")
		return
	}

	data := struct {
		Title    string
		ExtraCSS string
		ExtraJS  string
		Location data.Location
		Artists  []data.Artist
	}{
		Title:    fmt.Sprintf("%s - Location", location.Name),
		ExtraCSS: "location_detail.css",
		ExtraJS:  "",
		Location: location,
		Artists:  location.Artists,
	}

	h.render(w, r, "location_detail.tmpl", data)
}

// DevIndex renders a small developer page with quick links.
func (h *Handler) DevIndex(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.Error(w, r, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	links := []struct{ Href, Text string }{
		{"/dev/panic", "Trigger Panic (/dev/panic)"},
		{"/dev/404", "Simulate 404 (/dev/404)"},
		{"/dev/500", "Simulate 500 (/dev/500)"},
		{"/dev/tmpl-error", "Simulate Template Error (/dev/tmpl-error)"},
		{"/health", "Health Check (/health)"},
	}

	data := struct {
		Title    string
		ExtraCSS string
		ExtraJS  string
		Links    []struct{ Href, Text string }
	}{
		Title:    "Developer Tools",
		ExtraCSS: "dev.css",
		ExtraJS:  "",
		Links:    links,
	}

	h.render(w, r, "dev.tmpl", data)
}

// Error handles all errors (4xx and 5xx) in a centralized way.
func (h *Handler) Error(w http.ResponseWriter, r *http.Request, status int, message string) {
	data := struct {
		Title        string
		ExtraCSS     string
		ExtraJS      string
		ErrorCode    int
		RequestedURL string
		Message      string
		Timestamp    string
	}{
		Title:        fmt.Sprintf("%d %s", status, http.StatusText(status)),
		ExtraCSS:     "errors.css",
		ExtraJS:      "",
		ErrorCode:    status,
		RequestedURL: r.URL.Path,
		Message:      message,
		Timestamp:    time.Now().Format("2006-01-02 15:04:05"),
	}

	h.render(w, r, "error.tmpl", data, status)
}

// Health provides a health check endpoint.
func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		h.Error(w, r, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	response := map[string]any{
		"status":    "healthy",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"stats":     h.repo.GetStats(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// DevPanic is a development endpoint to test panic recovery.
func (h *Handler) DevPanic(w http.ResponseWriter, r *http.Request) {
	panic("Development panic triggered")
}

// Dev404 is a development endpoint to test 404 error template.
func (h *Handler) Dev404(w http.ResponseWriter, r *http.Request) {
	// Simulate a realistic 404 by mutating a shallow copy of the request
	// so that template rendering sees a non-existent requested URL.
	// We keep the original request untouched and pass the modified copy
	// to the Home handler which will validate the path and trigger a 404.
	nr := new(http.Request)
	*nr = *r
	// Ensure method is GET and set a path that we know doesn't exist in the router
	nr.Method = http.MethodGet
	nr.URL.Path = "/this-page-does-not-exist"

	// Call Home with the modified request so the Error template is rendered
	// using the realistic requested URL stored in nr.URL.Path.
	h.Home(w, nr)
}

// Dev500 is a development endpoint to test 500 error template.
func (h *Handler) Dev500(w http.ResponseWriter, r *http.Request) {
	h.Error(w, r, http.StatusInternalServerError, "This is a simulated 500 error.")
}

// Dev500Tmpl is a development endpoint to test template failure.
func (h *Handler) Dev500Tmpl(w http.ResponseWriter, r *http.Request) {
	// To simulate a template error, we can try to render a template that doesn't exist.
	h.render(w, r, "nonexistent.tmpl", nil)
}

func (h *Handler) StaticFiles(w http.ResponseWriter, r *http.Request) {
	const staticDir = "static"

	// Verify static directory exists
	if _, err := os.Stat(staticDir); err != nil {
		h.Error(w, r, http.StatusInternalServerError, "Static directory not found")
		return
	}

	// Only allow safe HTTP methods
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.Header().Set("Allow", "GET, HEAD")
		h.Error(w, r, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	// Handle favicon.ico requests explicitly
	reqPath := r.URL.Path
	if reqPath == "/favicon.ico" {
		h.serveFavicon(w, r, staticDir)
		return
	}

	// Only allow requests that start with /static/
	const prefix = "/static/"
	if !strings.HasPrefix(reqPath, prefix) {
		h.Error(w, r, http.StatusNotFound, "Not found")
		return
	}

	// Extract and validate the relative path
	rel := strings.TrimPrefix(reqPath, prefix)
	if !h.isValidStaticPath(rel) {
		h.Error(w, r, http.StatusNotFound, "Not found")
		return
	}

	// Build the target file path securely
	target := filepath.Join(staticDir, rel)
	if !h.isPathSafe(staticDir, target) {
		h.Error(w, r, http.StatusNotFound, "Not found")
		return
	}

	// Verify the target is a regular file
	fi, err := os.Stat(target)
	if err != nil || fi.IsDir() {
		h.Error(w, r, http.StatusNotFound, "Not found")
		return
	}

	// Handle conditional requests (304 Not Modified)
	if h.handleConditionalRequest(w, r, fi) {
		return // 304 Not Modified was sent
	}

	// Set content type and caching headers
	h.setStaticFileHeaders(w, target)

	// Serve the file
	http.ServeFile(w, r, target)
}
