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

// AppData holds the application state and handlers.
type AppData struct {
	repo      *repository.Repository
	templates *template.Template
}

// NewHandler creates a new handler with the given repository.
func NewHandler(repo *repository.Repository) *AppData {
	h := &AppData{repo: repo}
	h.loadTemplates()
	return h
}

// Home handles the home page.
func (a *AppData) Home(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if r.URL.Path != "/" {
		a.NotFound(w, r)
		return
	}

	artists := a.repo.GetArtists()
	stats := a.repo.GetStats()
	locations := a.repo.GetLocations()

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

	a.render(w, r, "home.tmpl", data)
}

// Artists handles the artists listing page.
func (a *AppData) Artists(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if r.URL.Path != "/artists" {
		a.NotFound(w, r)
		return
	}

	artists := a.repo.GetArtists()

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

	a.render(w, r, "artists.tmpl", data)
}

// ArtistDetail handles individual artist pages.
func (a *AppData) ArtistDetail(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract artist identifier from URL
	path := strings.TrimPrefix(r.URL.Path, "/artists/")
	if path == "" {
		a.NotFound(w, r)
		return
	}

	var artist repository.Artist
	var found bool

	// Try slug first, then ID
	if artist, found = a.repo.GetArtistBySlug(path); !found {
		if id, err := strconv.Atoi(path); err == nil {
			artist, found = a.repo.GetArtist(id)
		}
	}

	if !found {
		a.NotFound(w, r)
		return
	}

	concert, _ := a.repo.GetConcert(artist.ID)
	prev, next := a.repo.GetNextPrevArtist(artist)

	data := struct {
		Title      string
		ExtraCSS   string
		ExtraJS    string
		Artist     repository.Artist
		Relation   repository.Concert
		TotalShows int
		Countries  []string
		PrevArtist *repository.Artist
		NextArtist *repository.Artist
	}{
		Title:      artist.Name,
		ExtraCSS:   "artist_detail.css",
		ExtraJS:    "",
		Artist:     artist,
		Relation:   concert, // Using "Relation" for template compatibility
		TotalShows: a.repo.CountShows(concert),
		Countries:  a.repo.GetCountries(concert),
		PrevArtist: prev,
		NextArtist: next,
	}

	a.render(w, r, "artist_detail.tmpl", data)
}

// Locations handles the locations listing page.
func (a *AppData) Locations(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if r.URL.Path != "/locations" {
		a.NotFound(w, r)
		return
	}

	locations := a.repo.GetLocations()
	locationStats := a.repo.GetLocationStats()
	globalStats := a.repo.GetStats()

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

	a.render(w, r, "locations.tmpl", data)
}

// LocationDetail handles individual location pages.
func (a *AppData) LocationDetail(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract location slug from URL
	slug := strings.TrimPrefix(r.URL.Path, "/locations/")
	if slug == "" {
		a.NotFound(w, r)
		return
	}

	location, found := a.repo.GetLocationBySlug(slug)
	if !found {
		a.NotFound(w, r)
		return
	}

	data := struct {
		Title        string
		ExtraCSS     string
		ExtraJS      string
		LocationName string
		DisplayName  string
		Location     repository.LocationStats
		Artists      []repository.Artist
	}{
		Title:        fmt.Sprintf("%s - Location", location.DisplayName),
		ExtraCSS:     "locations.css",
		ExtraJS:      "",
		LocationName: location.Name,
		DisplayName:  location.DisplayName,
		Location:     location,
		Artists:      location.Artists,
	}

	a.render(w, r, "location_detail.tmpl", data)
}

// Health provides a health check endpoint.
func (a *AppData) Health(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	response := map[string]interface{}{
		"status":    "healthy",
		"timestamp": r.Header.Get("X-Request-Time"),
		"stats":     a.repo.GetStats(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// NotFound handles 404 errors.
func (a *AppData) NotFound(w http.ResponseWriter, r *http.Request) {
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

	// Let render method handle both Content-Type and status
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusNotFound)
	if err := a.templates.ExecuteTemplate(w, "error.tmpl", data); err != nil {
		log.Printf("Template execution error for error.tmpl: %v", err)
		// Don't try to call render again, just log the error
	}
}

// InternalError handles 500 errors.
func (a *AppData) InternalError(w http.ResponseWriter, r *http.Request, message string) {
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

	// Handle status and template execution directly to avoid cycles
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusInternalServerError)
	if err := a.templates.ExecuteTemplate(w, "error.tmpl", data); err != nil {
		log.Printf("Template execution error for error.tmpl: %v", err)
		// Don't try to call render again, just log the error
	}
}

// DevPanic is a development endpoint to test panic recovery.
func (a *AppData) DevPanic(w http.ResponseWriter, r *http.Request) {
	panic("Development panic triggered")
}

// Private helper methods

func (a *AppData) loadTemplates() {
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
	a.templates, err = template.New("").Funcs(funcMap).ParseFiles(templateFiles...)
	if err != nil {
		log.Fatalf("Failed to parse templates: %v", err)
	}
}

func (a *AppData) render(w http.ResponseWriter, r *http.Request, templateName string, data interface{}) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := a.templates.ExecuteTemplate(w, templateName, data); err != nil {
		log.Printf("Template execution error for %s: %v", templateName, err)
		// Don't call InternalError as it would create a cycle
		// Only write error response if this isn't an error template already failing
		if templateName != "error.tmpl" {
			// Write a simple error message without setting headers again
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
