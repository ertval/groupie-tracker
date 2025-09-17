// Package handlers provides HTTP handlers for the Groupie Tracker web application.
package handlers

import (
	"encoding/json"
	"fmt"
	"groupie-tracker/internal/repository"
	"html/template"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Handler holds the application state and handlers.
type Handler struct {
	repo      *repository.Repository
	templates *template.Template
}

// NewHandler creates a new handler with the given repository.
func NewHandler(repo *repository.Repository) *Handler {
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
		Artists        []repository.Artist
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
		Artists  []repository.Artist
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

	var artist repository.Artist
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
		Title      string
		ExtraCSS   string
		ExtraJS    string
		Artist     repository.Artist
		TotalShows int
		Countries  []string
		PrevArtist *repository.Artist
		NextArtist *repository.Artist
	}{
		Title:      artist.Name,
		ExtraCSS:   "artist_detail.css",
		ExtraJS:    "",
		Artist:     artist,
		TotalShows: h.repo.CountShows(artist),
		Countries:  h.repo.GetCountries(artist),
		PrevArtist: prev,
		NextArtist: next,
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
		LocationStats  []repository.LocationStats
		TopLocations   []repository.LocationStats
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
		TotalConcerts:  globalStats["total_shows"],
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
		Location     repository.LocationStats
		Artists      []repository.Artist
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

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)

	if err := h.templates.ExecuteTemplate(w, templateName, data); err != nil {
		log.Printf("Template execution error for %s: %v", templateName, err)
		// Don't call InternalError as it would create a cycle
		// Only write error response if this isn't an error template already failing
		if templateName != "error.tmpl" {
			// Template failed, but headers already sent - log and write minimal error
			w.Write([]byte("Template rendering failed"))
		} else {
			// If error.tmpl itself fails, write plain text
			w.Write([]byte("Internal server error occurred"))
		}
	}
}

// createSlug creates a URL-friendly slug from a string.
func createSlug(input string) string {
	// Convert to lowercase and replace non-alphanumeric with hyphens
	reg := regexp.MustCompile(`[^a-zA-Z0-9]+`)
	slug := reg.ReplaceAllString(strings.ToLower(input), "-")
	return strings.Trim(slug, "-")
}
