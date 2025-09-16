// Package handlers provides HTTP request handlers for the Groupie Tracker application.
package handlers

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"groupie-tracker/internal/data"
)

// TemplateData provides a flexible way to pass data to templates
// while maintaining backward compatibility
type TemplateData map[string]any

// newTemplateData creates template data with common fields
func newTemplateData(title, extraCSS string, extraJS string) TemplateData {
	return TemplateData{
		"Title":    title,
		"ExtraCSS": extraCSS,
		"ExtraJS":  extraJS,
	}
}

// Handlers contains all HTTP handlers for the application.
type Handlers struct {
	repo      *data.Repository
	templates *template.Template
}

// NewHandlers creates a new handlers instance.
func NewHandlers(repo *data.Repository) *Handlers {
	h := &Handlers{
		repo: repo,
	}
	h.loadTemplates()
	return h
}

func (h *Handlers) loadTemplates() {
	templateFiles := []string{
		"templates/base.tmpl",
		"templates/home.tmpl",
		"templates/artists.tmpl",
		"templates/artist_detail.tmpl",
		"templates/locations.tmpl",
		"templates/location_detail.tmpl",
		"templates/error.tmpl",
	}

	funcMap := template.FuncMap{
		"add": func(a, b int) int { return a + b },
		"sub": func(a, b int) int { return a - b },
		"join": func(items []string, sep string) string {
			return strings.Join(items, sep)
		},
		"generateLocationSlug":  data.GenerateLocationSlug,
		"normalizeLocationName": data.NormalizeLocationName,
	}

	var err error
	h.templates, err = template.New("").Funcs(funcMap).ParseFiles(templateFiles...)
	if err != nil {
		log.Fatalf("Failed to parse templates: %v", err)
	}
}

// HomeHandler handles the home page.
func (h *Handlers) HomeHandler(w http.ResponseWriter, r *http.Request) {
	if !h.validateMethod(w, r, http.MethodGet) {
		return
	}

	if h.validatePath(w, r, "/") == "" {
		return // 404 already handled by validatePath
	}

	artists := h.repo.GetAllArtistsSorted()
	locations := h.repo.GetUniqueLocations()

	data := newTemplateData("Home", "home.css", "")
	data["Artists"] = artists
	data["TotalMembers"] = h.repo.GetTotalMembers()
	data["TotalLocations"] = len(locations)

	h.executeTemplate(w, r, "home.tmpl", data)
}

// ArtistsHandler handles requests to /artists page.
func (h *Handlers) ArtistsHandler(w http.ResponseWriter, r *http.Request) {
	if !h.validateMethod(w, r, http.MethodGet) {
		return
	}

	if h.validatePath(w, r, "/artists") == "" {
		return // 404 already handled by validatePath
	}

	artists := h.repo.GetAllArtistsSorted()
	data := newTemplateData("Artists", "artists.css", "")
	data["Artists"] = artists

	h.executeTemplate(w, r, "artists.tmpl", data)
}

// ArtistDetailHandler handles requests to specific artist pages.
func (h *Handlers) ArtistDetailHandler(w http.ResponseWriter, r *http.Request) {
	if !h.validateMethod(w, r, http.MethodGet) {
		return
	}

	// Extract artist identifier from path using parameterized validation
	identifier := h.validatePath(w, r, "/artists/")
	if identifier == "" {
		return // 404 already handled by validatePath
	}

	var artist data.Artist
	var found bool

	// Try to get artist by slug first (SEO-friendly URLs)
	artist, found = h.repo.GetArtistBySlug(identifier)
	if !found {
		// If slug doesn't work, try parsing as ID
		if id, err := strconv.Atoi(identifier); err == nil {
			artist, found = h.repo.GetArtist(id)
		}
	}

	if !found {
		h.NotFoundHandler(w, r)
		return
	}

	relation, _ := h.repo.GetRelation(artist.ID)
	prevArtist, nextArtist := h.repo.GetArtistNavigation(artist)

	data := newTemplateData(artist.Name, "artist_detail.css", "")
	data["Artist"] = artist
	data["Relation"] = relation
	data["PrevArtist"] = prevArtist
	data["NextArtist"] = nextArtist
	data["TotalShows"] = h.repo.CalculateTotalShows(relation)
	data["Countries"] = h.repo.ExtractCountries(relation)

	h.executeTemplate(w, r, "artist_detail.tmpl", data)
}

// LocationsHandler handles the locations page.
func (h *Handlers) LocationsHandler(w http.ResponseWriter, r *http.Request) {
	if !h.validateMethod(w, r, http.MethodGet) {
		return
	}

	if h.validatePath(w, r, "/locations") == "" {
		return // 404 already handled by validatePath
	}

	locations := h.repo.GetUniqueLocations()
	locationStats := h.repo.CalculateLocationStats()

	data := newTemplateData("Locations", "locations.css", "")
	data["Locations"] = locations
	data["LocationStats"] = locationStats
	data["TopLocations"] = locationStats
	data["TotalCountries"] = h.repo.GetTotalCountries()
	data["TotalConcerts"] = h.repo.GetStats()["total_concerts"]

	h.executeTemplate(w, r, "locations.tmpl", data)
}

// LocationDetailHandler handles requests to specific location pages.
func (h *Handlers) LocationDetailHandler(w http.ResponseWriter, r *http.Request) {
	if !h.validateMethod(w, r, http.MethodGet) {
		return
	}

	// Extract location slug from path using parameterized validation
	locationSlug := h.validatePath(w, r, "/locations/")
	if locationSlug == "" {
		return // 404 already handled by validatePath
	}
	locationDetail, found := h.repo.GetLocationDetailsBySlug(locationSlug)
	if !found {
		h.NotFoundHandler(w, r)
		return
	}

	artistsWithDates := h.repo.GetArtistsWithDatesForLocation(locationDetail.Name)

	data := newTemplateData(fmt.Sprintf("%s - Location", locationDetail.DisplayName), "locations.css", "")
	data["LocationName"] = locationDetail.Name
	data["DisplayName"] = locationDetail.DisplayName
	data["Artists"] = locationDetail.Artists
	data["ArtistsWithDates"] = artistsWithDates
	data["ConcertDates"] = locationDetail.Dates
	data["ArtistCount"] = locationDetail.ArtistCount
	data["ConcertCount"] = locationDetail.ConcertCount

	h.executeTemplate(w, r, "location_detail.tmpl", data)
}

// HealthHandler handles health check requests.
func (h *Handlers) HealthHandler(w http.ResponseWriter, r *http.Request) {
	if !h.validateMethod(w, r, http.MethodGet) {
		return
	}

	stats := h.repo.GetStats()
	status := "ok"
	if stats["artists"] == 0 {
		status = "error"
	}

	response := struct {
		Status string         `json:"status"`
		Stats  map[string]int `json:"stats"`
	}{
		Status: status,
		Stats:  stats,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Error encoding health response: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// NotFoundHandler handles 404 errors.
func (h *Handlers) NotFoundHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotFound)

	data := newTemplateData("Page Not Found", "errors.css", "")
	data["Message"] = "The page you're looking for doesn't exist."
	data["ErrorCode"] = 404
	data["RequestedURL"] = r.URL.Path

	h.executeTemplate(w, r, "error.tmpl", data)
}

// InternalErrorHandler handles 500 errors.
func (h *Handlers) InternalErrorHandler(w http.ResponseWriter, r *http.Request, message string) {
	log.Printf("Internal error: %s", message)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusInternalServerError)

	if h.templates != nil {
		data := newTemplateData("Internal Server Error", "errors.css", "")
		data["Message"] = "Something went wrong on our end. We're working to fix it!"
		data["ErrorCode"] = 500
		data["RequestedURL"] = r.URL.Path
		data["Timestamp"] = time.Now().Format("2006-01-02 15:04:05")

		if err := h.templates.ExecuteTemplate(w, "error.tmpl", data); err != nil {
			log.Printf("Template execution error: %v", err)
			h.writeSimpleHTML(w, "Internal Server Error", "An error occurred while rendering the page.")
		}
	} else {
		h.writeSimpleHTML(w, "Internal Server Error", "An error occurred and templates are not available.")
	}
}

func (h *Handlers) writeSimpleHTML(w http.ResponseWriter, title, content string) {
	html := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
    <title>%s - Groupie Tracker</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 40px; }
        h1 { color: #333; }
    </style>
</head>
<body>
    <h1>%s</h1>
    <p>%s</p>
    <p><a href="/">← Back to Home</a></p>
</body>
</html>`, title, title, content)

	fmt.Fprint(w, html)
}

func (h *Handlers) validateMethod(w http.ResponseWriter, r *http.Request, expectedMethod string) bool {
	if r.Method != expectedMethod {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return false
	}
	return true
}

func (h *Handlers) executeTemplate(w http.ResponseWriter, r *http.Request, templateName string, data interface{}) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if h.templates != nil {
		if err := h.templates.ExecuteTemplate(w, templateName, data); err != nil {
			log.Printf("Template execution error: %v", err)
			h.InternalErrorHandler(w, r, fmt.Sprintf("Template error: %v", err))
		}
	} else {
		log.Printf("Templates not available for %s", templateName)
		w.WriteHeader(http.StatusInternalServerError)
		h.writeSimpleHTML(w, "Template Error", "Templates are not available.")
	}
}

// PanicHandler is a dev/test handler that intentionally panics.
func (h *Handlers) PanicHandler(w http.ResponseWriter, r *http.Request) {
	panic("This is an intentional panic for testing the recovery middleware")
}

// validatePath handles both exact path matching and parameterized path validation.
// For exact paths (e.g., "/", "/artists"): returns "*" if valid, "" if invalid
// For parameterized paths (e.g., "/artists/"): returns parameter if valid, "" if invalid
// Always handles 404 response automatically for invalid paths.
func (h *Handlers) validatePath(w http.ResponseWriter, r *http.Request, expectedPath string) string {
	// Special case: root path "/" should be treated as exact match
	if expectedPath == "/" {
		if r.URL.Path == "/" {
			return "*" // Special value indicating exact match success
		}
		h.NotFoundHandler(w, r)
		return ""
	}

	// Handle exact path matching (doesn't end with "/" except for root)
	if !strings.HasSuffix(expectedPath, "/") {
		if r.URL.Path == expectedPath {
			return "*" // Special value indicating exact match success
		}
		h.NotFoundHandler(w, r)
		return ""
	}

	// Handle parameterized paths (ending with "/")
	pathParts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	expectedParts := strings.Split(strings.Trim(expectedPath, "/"), "/")

	if len(pathParts) != len(expectedParts)+1 { // +1 for the parameter
		h.NotFoundHandler(w, r)
		return ""
	}

	// Check that the base path matches
	for i, expected := range expectedParts {
		if pathParts[i] != expected {
			h.NotFoundHandler(w, r)
			return ""
		}
	}

	// Return the parameter (last part of the path)
	return pathParts[len(pathParts)-1]
}
