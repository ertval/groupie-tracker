package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
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
	if r.Method != http.MethodGet {
		h.Error(w, r, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	if r.URL.Path != "/" {
		h.Error(w, r, http.StatusNotFound, "Page not found")
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
	if r.Method != http.MethodGet {
		h.Error(w, r, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	if r.URL.Path != "/artists" {
		h.Error(w, r, http.StatusNotFound, "Page not found")
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

	var artist data.Artist
	var found bool

	if artist, found = h.repo.GetArtistBySlug(path); !found {
		id, err := strconv.Atoi(path)
		if err != nil {
			h.Error(w, r, http.StatusNotFound, "Artist not found")
			return
		}
		if artist, found = h.repo.GetArtistByID(id); !found {
			h.Error(w, r, http.StatusNotFound, "Artist not found")
			return
		}
	}

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
	if r.Method != http.MethodGet {
		h.Error(w, r, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	if r.URL.Path != "/locations" {
		h.Error(w, r, http.StatusNotFound, "Page not found")
		return
	}

	locations := h.repo.GetLocations()
	globalStats := h.repo.GetStats()

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
		TotalCountries: globalStats["total_countries"],
		TotalConcerts:  globalStats["total_concerts"],
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

// DevIndex renders a small developer page with quick links to the dev handlers.
func (h *Handler) DevIndex(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.Error(w, r, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	links := []struct{ Href, Text string }{
		{Href: "/dev/panic", Text: "Trigger Panic (/dev/panic)"},
		{Href: "/dev/404", Text: "Simulate 404 (/dev/404)"},
		{Href: "/dev/500", Text: "Simulate 500 (/dev/500)"},
		{Href: "/dev/tmpl-error", Text: "Simulate Template Error (/dev/tmpl-error)"},
		{Href: "/health", Text: "Health Check (/health)"},
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

	// Special case for rendering errors to avoid recursion.
	ts, ok := h.templates["error.tmpl"]
	if !ok {
		// Fallback if the error template itself is not found
		http.Error(w, "500 Internal Server Error - Error template not found (http.Error returned)", http.StatusInternalServerError)
		return
	}

	buf := new(bytes.Buffer)
	err := ts.ExecuteTemplate(buf, "base", data)
	if err != nil {
		// Fallback if executing the error template fails
		log.Printf("Error executing error template: %v", err)
		http.Error(w, "500 Internal Server Error - Failed to execute error template", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(status)
	buf.WriteTo(w)
}

// DevPanic is a development endpoint to test panic recovery.
func (h *Handler) DevPanic(w http.ResponseWriter, r *http.Request) {
	panic("Development panic triggered")
}

// Dev404 is a development endpoint to test 404 error template.
func (h *Handler) Dev404(w http.ResponseWriter, r *http.Request) {
	// Simulate a proper 404 error - this tests if the error template renders correctly
	h.Error(w, r, http.StatusNotFound, "This is a simulated 404 error.")
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

	if _, err := os.Stat(staticDir); err != nil {
		h.Error(w, r, http.StatusInternalServerError, "Static directory not found")
		return
	}

	// Only allow safe methods
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.Header().Set("Allow", "GET, HEAD")
		h.Error(w, r, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	// Normalize and validate the path to avoid path traversal
	reqPath := r.URL.Path
	if reqPath == "/favicon.ico" {
		// Serve favicon explicitly
		faviconPath := filepath.Join(staticDir, "favicon.ico")
		if fi, err := os.Stat(faviconPath); err != nil || fi.IsDir() {
			h.Error(w, r, http.StatusNotFound, "favicon not found")
			return
		}
		http.ServeFile(w, r, faviconPath)
		return
	}

	// Only allow requests that start with /static/
	const prefix = "/static/"
	if !strings.HasPrefix(reqPath, prefix) {
		h.Error(w, r, http.StatusNotFound, "Not found")
		return
	}

	// Clean the path and prevent directory listing
	rel := strings.TrimPrefix(reqPath, prefix)
	rel = filepath.Clean(rel)
	if rel == "." || rel == "" || strings.HasSuffix(reqPath, "/") {
		// don't allow directory browsing
		h.Error(w, r, http.StatusNotFound, "Not found")
		return
	}

	// Prevent path traversal: ensure the resulting path is within staticDir
	target := filepath.Join(staticDir, rel)
	absStatic, _ := filepath.Abs(staticDir)
	absTarget, err := filepath.Abs(target)
	if err != nil || !strings.HasPrefix(absTarget, absStatic+string(os.PathSeparator)) && absTarget != absStatic {
		h.Error(w, r, http.StatusNotFound, "Not found")
		return
	}

	// Ensure the target is a regular file
	fi, err := os.Stat(target)
	if err != nil || fi.IsDir() {
		h.Error(w, r, http.StatusNotFound, "Not found")
		return
	}

	// Serve the file
	http.ServeFile(w, r, target)
}

// --- Private Helper Methods ---

// render renders a template with the given data and status code.
func (h *Handler) render(w http.ResponseWriter, r *http.Request, name string, data any, status ...int) {
	code := http.StatusOK
	if len(status) > 0 {
		code = status[0]
	}

	tmpl, ok := h.templates[name]
	if !ok {
		h.Error(w, r, http.StatusInternalServerError, fmt.Sprintf("Template %s not found", name))
		return
	}

	buf := new(bytes.Buffer)
	if err := tmpl.ExecuteTemplate(buf, "base", data); err != nil {
		h.Error(w, r, http.StatusInternalServerError, err.Error())
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(code)
	buf.WriteTo(w)
}

// loadTemplates loads and parses all templates from the templates directory.
func (h *Handler) loadTemplates() {
	h.templates = make(map[string]*template.Template)

	funcMap := template.FuncMap{
		"add":   func(a, b int) int { return a + b },
		"sub":   func(a, b int) int { return a - b },
		"join":  func(items []string, sep string) string { return strings.Join(items, sep) },
		"upper": func(s string) string { return strings.ToUpper(s) },
		"title": func(s string) string {
			words := strings.Fields(strings.ReplaceAll(s, "-", " "))
			for i, word := range words {
				if len(word) > 0 {
					words[i] = strings.ToUpper(string(word[0])) + strings.ToLower(word[1:])
				}
			}
			return strings.Join(words, " ")
		},
	}

	const templateDir = "templates"
	baseTmplPath := filepath.Join(templateDir, "base.tmpl")

	if _, err := os.Stat(baseTmplPath); err != nil {
		log.Fatalf("Failed to find base template at %s: %v", baseTmplPath, err)
	}

	pages, err := filepath.Glob(filepath.Join(templateDir, "*.tmpl"))
	if err != nil {
		log.Fatalf("Failed to glob templates: %v", err)
	}

	for _, page := range pages {
		name := filepath.Base(page)
		if name == "base.tmpl" {
			continue
		}

		ts, err := template.New(name).Funcs(funcMap).ParseFiles(baseTmplPath, page)
		if err != nil {
			log.Fatalf("Failed to parse template %s: %v", name, err)
		}

		h.templates[name] = ts
	}
}
