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

	h.render(w, r, "error.tmpl", data, status)
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
	h.setStaticFileHeaders(w, target, fi)

	// Serve the file
	http.ServeFile(w, r, target)
}

// generateETag creates a simple ETag based on file size and modification time
func (h *Handler) generateETag(fi os.FileInfo) string {
	return fmt.Sprintf(`"%x-%x"`, fi.Size(), fi.ModTime().Unix())
}

// getContentType returns the appropriate content type for file extensions
func (h *Handler) getContentType(ext string) string {
	switch ext {
	case ".css":
		return "text/css; charset=utf-8"
	case ".js":
		return "application/javascript; charset=utf-8"
	case ".png":
		return "image/png"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".gif":
		return "image/gif"
	case ".svg":
		return "image/svg+xml"
	case ".ico":
		return "image/x-icon"
	case ".woff":
		return "font/woff"
	case ".woff2":
		return "font/woff2"
	case ".ttf":
		return "font/ttf"
	case ".eot":
		return "application/vnd.ms-fontobject"
	default:
		return "application/octet-stream"
	}
}

// getCacheControl returns appropriate cache control headers based on file type
func (h *Handler) getCacheControl(ext string) string {
	switch ext {
	case ".css", ".js":
		// CSS and JS files - cache for 1 year
		return "public, max-age=31536000"
	case ".png", ".jpg", ".jpeg", ".gif", ".svg", ".ico":
		// Images - cache for 1 month
		return "public, max-age=2592000"
	case ".woff", ".woff2", ".ttf", ".eot":
		// Fonts - cache for 1 year
		return "public, max-age=31536000"
	default:
		// Other files - cache for 1 hour
		return "public, max-age=3600"
	}
}

// handleConditionalRequest handles If-None-Match and If-Modified-Since headers
func (h *Handler) handleConditionalRequest(w http.ResponseWriter, r *http.Request, fi os.FileInfo) bool {
	modTime := fi.ModTime()
	etag := h.generateETag(fi)

	// Set ETag and Last-Modified headers
	w.Header().Set("ETag", etag)
	w.Header().Set("Last-Modified", modTime.UTC().Format(http.TimeFormat))

	// Check If-None-Match (ETag)
	if inm := r.Header.Get("If-None-Match"); inm != "" {
		if inm == etag || inm == "*" {
			w.WriteHeader(http.StatusNotModified)
			return true
		}
	}

	// Check If-Modified-Since
	if ims := r.Header.Get("If-Modified-Since"); ims != "" {
		if t, err := http.ParseTime(ims); err == nil {
			// Compare with 1-second precision (HTTP time format limitation)
			if modTime.Unix() <= t.Unix() {
				w.WriteHeader(http.StatusNotModified)
				return true
			}
		}
	}

	return false
}

// setStaticFileHeaders sets appropriate headers for static files
func (h *Handler) setStaticFileHeaders(w http.ResponseWriter, target string, fi os.FileInfo) {
	// Set content type based on file extension
	ext := strings.ToLower(filepath.Ext(target))
	contentType := h.getContentType(ext)
	w.Header().Set("Content-Type", contentType)

	// Set caching headers based on file type
	cacheControl := h.getCacheControl(ext)
	w.Header().Set("Cache-Control", cacheControl)
	w.Header().Set("Vary", "Accept-Encoding")

	// Set security headers for static files
	w.Header().Set("X-Content-Type-Options", "nosniff")
}

// isValidStaticPath validates the relative path to prevent directory traversal
func (h *Handler) isValidStaticPath(rel string) bool {
	// Clean the path and normalize path separators
	rel = filepath.Clean(rel)
	rel = filepath.ToSlash(rel) // Convert Windows backslashes to forward slashes

	// Reject empty, current directory, or paths ending with slash
	if rel == "." || rel == "" || strings.HasSuffix(rel, "/") {
		return false
	}

	// Reject paths that try to go up directories
	if strings.Contains(rel, "..") {
		return false
	}

	// Reject paths starting with slash
	if strings.HasPrefix(rel, "/") {
		return false
	}

	return true
}

// isPathSafe ensures the resolved path is within the static directory
func (h *Handler) isPathSafe(staticDir, target string) bool {
	absStatic, err := filepath.Abs(staticDir)
	if err != nil {
		return false
	}

	absTarget, err := filepath.Abs(target)
	if err != nil {
		return false
	}

	// Ensure target is within static directory
	staticPrefix := absStatic + string(filepath.Separator)
	return strings.HasPrefix(absTarget, staticPrefix) || absTarget == absStatic
}

// serveFavicon handles favicon.ico requests with appropriate caching
func (h *Handler) serveFavicon(w http.ResponseWriter, r *http.Request, staticDir string) {
	faviconPath := filepath.Join(staticDir, "favicon.ico")
	fi, err := os.Stat(faviconPath)
	if err != nil || fi.IsDir() {
		h.Error(w, r, http.StatusNotFound, "Favicon not found")
		return
	}

	// Handle conditional requests for favicon
	if h.handleConditionalRequest(w, r, fi) {
		return
	}

	// Set favicon-specific headers
	w.Header().Set("Cache-Control", "public, max-age=86400") // 24 hours for favicon
	w.Header().Set("Vary", "Accept-Encoding")
	w.Header().Set("X-Content-Type-Options", "nosniff")

	http.ServeFile(w, r, faviconPath)
}

// --- Private Helper Methods ---

// render renders a template with the given data and status code (if provided) managing all errors.
func (h *Handler) render(w http.ResponseWriter, r *http.Request, name string, data any, status ...int) {
	code := http.StatusOK
	if len(status) > 0 {
		code = status[0]
	}

	tmpl, ok := h.templates[name]
	if !ok {
		// If we are already rendering an error, don't call h.Error again.
		if name == "error.tmpl" {
			log.Printf("FATAL: error.tmpl is missing")
			http.Error(w, "500 Internal Server Error - Error template not found", http.StatusInternalServerError)
			return
		}
		h.Error(w, r, http.StatusInternalServerError, fmt.Sprintf("Template %s not found", name))
		return
	}

	buf := new(bytes.Buffer)
	if err := tmpl.ExecuteTemplate(buf, "base", data); err != nil {
		// If we get an error while executing the error template, we need to fallback.
		if name == "error.tmpl" {
			log.Printf("Error executing error template: %v", err)
			http.Error(w, "500 Internal Server Error - Failed to execute error template", http.StatusInternalServerError)
			return
		}
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
