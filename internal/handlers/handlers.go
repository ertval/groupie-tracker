// Package handlers provides HTTP handlers for the Groupie Tracker web application.
package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"groupie-tracker/internal/data"
	"html/template"
	"log"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Handler holds the application state and handlers.
type Handler struct {
	repo      *data.Repository
	templates *template.Template
}

// NewHandler creates a new handler with the given repository.
func NewHandler(repo *data.Repository) *Handler {
	h := &Handler{repo: repo}
	h.loadTemplates()
	return h
}

// Home handles the home page.
func (h *Handler) Home(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if r.URL.Path != "/" {
		h.NotFound(w, r)
		return
	}

	artists := h.repo.GetArtists()
	stats := h.repo.GetStats()
	locations := h.repo.GetLocations()

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
		TotalLocations: len(locations),
	}

	h.render(w, "home.tmpl", data)
}

// Artists handles the artists listing page.
func (h *Handler) Artists(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if r.URL.Path != "/artists" {
		h.NotFound(w, r)
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

	h.render(w, "artists.tmpl", data)
}

// ArtistDetail handles individual artist pages.
func (h *Handler) ArtistDetail(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract artist identifier from URL
	path := strings.TrimPrefix(r.URL.Path, "/artists/")
	if path == "" {
		h.NotFound(w, r)
		return
	}

	var artist data.Artist
	var found bool

	// Try slug first, then ID
	if artist, found = h.repo.GetArtistBySlug(path); !found {
		if id, err := strconv.Atoi(path); err == nil {
			artist, found = h.repo.GetArtist(id)
		}
	}

	if !found {
		h.NotFound(w, r)
		return
	}

	prev, next := h.repo.GetNextPrevArtist(artist)

	data := struct {
		Title         string
		ExtraCSS      string
		ExtraJS       string
		Artist        data.Artist
		TotalConcerts int
		Countries     []string
		PrevArtist    *data.Artist
		NextArtist    *data.Artist
	}{
		Title:         artist.Name,
		ExtraCSS:      "artist_detail.css",
		ExtraJS:       "",
		Artist:        artist,
		TotalConcerts: h.repo.CountConcerts(artist),
		Countries:     h.repo.GetCountries(artist),
		PrevArtist:    prev,
		NextArtist:    next,
	}

	h.render(w, "artist_detail.tmpl", data)
}

// Locations handles the locations listing page.
func (h *Handler) Locations(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if r.URL.Path != "/locations" {
		h.NotFound(w, r)
		return
	}

	locations := h.repo.GetLocations()
	locationStats := h.repo.GetLocationStats()
	globalStats := h.repo.GetStats()

	data := struct {
		Title          string
		ExtraCSS       string
		ExtraJS        string
		Locations      []string
		LocationStats  []data.LocationStats
		TopLocations   []data.LocationStats
		TotalCountries int
		TotalConcerts  int
	}{
		Title:          "Locations",
		ExtraCSS:       "locations.css",
		ExtraJS:        "",
		Locations:      locations,
		LocationStats:  locationStats,
		TopLocations:   locationStats, // Same data for template compatibility
		TotalCountries: globalStats["total_countries"],
		TotalConcerts:  globalStats["total_concerts"],
	}

	h.render(w, "locations.tmpl", data)
}

// LocationDetail handles individual location pages.
func (h *Handler) LocationDetail(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract location slug from URL
	slug := strings.TrimPrefix(r.URL.Path, "/locations/")
	if slug == "" {
		h.NotFound(w, r)
		return
	}

	location, found := h.repo.GetLocationBySlug(slug)
	if !found {
		h.NotFound(w, r)
		return
	}

	data := struct {
		Title        string
		ExtraCSS     string
		ExtraJS      string
		LocationName string
		Location     data.LocationStats
		Artists      []data.Artist
	}{
		Title:        fmt.Sprintf("%s - Location", location.Name),
		ExtraCSS:     "locations.css",
		ExtraJS:      "",
		LocationName: location.Name,
		Location:     location,
		Artists:      location.Artists,
	}

	h.render(w, "location_detail.tmpl", data)
}

// Health provides a health check endpoint.
func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	response := map[string]any{
		"status":    "healthy",
		"timestamp": r.Header.Get("X-Request-Time"),
		"stats":     h.repo.GetStats(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// StaticFiles handles all static file requests with 404 fallback for missing assets.
func (h *Handler) StaticFiles(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var path string

	// Handle different static file request patterns
	switch {
	case r.URL.Path == "/favicon.ico":
		// Special handling for favicon requests
		path = "static/favicon.ico"
	case strings.HasPrefix(r.URL.Path, "/static/"):
		// Handle /static/ prefixed requests
		path = "static/" + strings.TrimPrefix(r.URL.Path, "/static/")
	default:
		// For any other static file patterns, treat as 404
		http.Error(w, "Not Found", http.StatusNotFound)
		return
	}

	// Check if the file exists
	if _, err := os.Stat(path); err != nil {
		// File doesn't exist - return 404 for all missing assets
		http.Error(w, "Not Found", http.StatusNotFound)
		return
	}

	// Serve the file
	http.ServeFile(w, r, path)
}

// NotFound handles 404 errors.
func (h *Handler) NotFound(w http.ResponseWriter, r *http.Request) {
	data := struct {
		Title        string
		ExtraCSS     string
		ExtraJS      string
		ErrorCode    int
		RequestedURL string
		Message      string
		Timestamp    string
	}{
		Title:        "Page Not Found",
		ExtraCSS:     "errors.css",
		ExtraJS:      "",
		ErrorCode:    404,
		RequestedURL: r.URL.Path,
		Message:      "The page you're looking for doesn't exist.",
		Timestamp:    time.Now().Format("2006-01-02 15:04:05"),
	}

	h.render(w, "error.tmpl", data, http.StatusNotFound)
}

// InternalError handles 500 errors.
func (h *Handler) InternalError(w http.ResponseWriter, r *http.Request, message string) {
	data := struct {
		Title        string
		ExtraCSS     string
		ExtraJS      string
		ErrorCode    int
		RequestedURL string
		Message      string
		Timestamp    string
	}{
		Title:        "Internal Server Error",
		ExtraCSS:     "errors.css",
		ExtraJS:      "",
		ErrorCode:    500,
		RequestedURL: r.URL.Path,
		Message:      message,
		Timestamp:    time.Now().Format("2006-01-02 15:04:05"),
	}

	h.render(w, "error.tmpl", data, http.StatusInternalServerError)
}

// DevPanic is a development endpoint to test panic recovery.
func (h *Handler) DevPanic(w http.ResponseWriter, r *http.Request) {
	panic("Development panic triggered")
}

// Dev404 is a development endpoint to test 404 error template.
func (h *Handler) Dev404(w http.ResponseWriter, r *http.Request) {
	// Simulate a client requesting a non-existent path so that the
	// normal handler flow produces a 404 via NotFound.
	req := r.Clone(r.Context())
	req.URL.Path = "/__does_not_exist__"
	h.Home(w, req)
}

// Dev500 is a development endpoint to test 500 error template.
func (h *Handler) Dev500(w http.ResponseWriter, r *http.Request) {
	// Simulate a normal handler encountering an internal error and
	// calling InternalError as part of its flow. We create a small
	// local mux and handler to represent that handler and then
	// dispatch a synthetic request through it.
	mux := http.NewServeMux()
	mux.HandleFunc("/cause-500", func(w http.ResponseWriter, r *http.Request) {
		// This represents an application handler that detected an error
		// and uses the standard InternalError path to render a response.
		h.InternalError(w, r, "Development 500 error triggered")
	})

	req := r.Clone(r.Context())
	req.URL.Path = "/cause-500"
	mux.ServeHTTP(w, req)
}

// DevTemplateError is a development endpoint to test template failure.
func (h *Handler) DevTemplateError(w http.ResponseWriter, r *http.Request) {
	// Simulate a template execution error by providing a templates set
	// that does not contain the template name. We create an empty
	// template set (non-nil) so ExecuteTemplate returns an undefined
	// template error when handlers call it.
	old := h.templates
	h.templates = template.New("")
	defer func() { h.templates = old }()

	req := r.Clone(r.Context())
	req.URL.Path = "/artists"
	h.Artists(w, req)
}

// DevIndex renders a small developer page with quick links to the dev
// handlers so developers can click to exercise panic/500/404/template errors.
func (h *Handler) DevIndex(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	links := []struct{ Href, Text string }{
		{Href: "/dev/panic", Text: "Trigger Panic (/dev/panic)"},
		{Href: "/dev/404", Text: "Simulate 404 (/dev/404)"},
		{Href: "/dev/500", Text: "Simulate 500 (/dev/500)"},
		{Href: "/dev/template-error", Text: "Simulate Template Error (/dev/template-error)"},
		{Href: "/health", Text: "Health Check (/health)"},
	}

	data := struct {
		Title    string
		ExtraCSS string
		ExtraJS  string
		Links    []struct{ Href, Text string }
	}{
		Title:    "Developer Tools",
		ExtraCSS: "home.css",
		ExtraJS:  "",
		Links:    links,
	}

	h.render(w, "dev.tmpl", data)
}

// Private helper methods

func (h *Handler) loadTemplates() {
	templateFiles := []string{
		"templates/base.tmpl",
		"templates/home.tmpl",
		"templates/artists.tmpl",
		"templates/artist_detail.tmpl",
		"templates/locations.tmpl",
		"templates/location_detail.tmpl",
		"templates/error.tmpl",
		"templates/dev.tmpl",
	}

	funcMap := template.FuncMap{
		"add": func(a, b int) int { return a + b },
		"sub": func(a, b int) int { return a - b },
		"join": func(items []string, sep string) string {
			return strings.Join(items, sep)
		},
		"generateLocationSlug": func(location string) string {
			return createSlug(location)
		},
		"normalizeLocationName": func(location string) string {
			// Manual title case implementation to replace deprecated strings.Title
			words := strings.Fields(strings.ReplaceAll(location, "_", " "))
			for i, word := range words {
				if len(word) > 0 {
					words[i] = strings.ToUpper(string(word[0])) + strings.ToLower(word[1:])
				}
			}
			return strings.Join(words, " ")
		},
	}

	var err error
	h.templates, err = template.New("").Funcs(funcMap).ParseFiles(templateFiles...)
	if err != nil {
		log.Fatalf("Failed to parse templates: %v", err)
	}
}

func (h *Handler) render(w http.ResponseWriter, templateName string, data any, statusCode ...int) {
	// Default to 200 OK if no status code provided
	status := http.StatusOK
	if len(statusCode) > 0 {
		status = statusCode[0]
	}

	// Use a buffer to run template execution before writing headers/body.
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	if h.templates == nil {
		// Templates are required for rendering; return 500 to signal a
		// server-side configuration/template issue.
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("500 Internal Server Error - Template Issue"))
		return
	}

	var buf bytes.Buffer
	if err := h.templates.ExecuteTemplate(&buf, templateName, data); err != nil {
		log.Printf("Template execution error for %s: %v", templateName, err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("500 Internal Server Error - Template Issue"))
		return
	}

	w.WriteHeader(status)
	w.Write(buf.Bytes())
}

// createSlug creates a URL-friendly slug from a string.
func createSlug(input string) string {
	// Convert to lowercase and replace non-alphanumeric with hyphens
	reg := regexp.MustCompile(`[^a-zA-Z0-9]+`)
	slug := reg.ReplaceAllString(strings.ToLower(input), "-")
	return strings.Trim(slug, "-")
}

// (years extraction moved to repository — templates now read precomputed ConcertYears)
