// Package handlers provides HTTP handlers for the Groupie Tracker web application.
package handlers

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"groupie-tracker/internal/app"
)

// Server holds the application state and handlers.
type Server struct {
	store     *app.Store
	templates *template.Template
}

// NewServer creates a new server with the given store.
func NewServer(store *app.Store) *Server {
	s := &Server{store: store}
	s.loadTemplates()
	return s
}

// Home handles the home page.
func (s *Server) Home(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if r.URL.Path != "/" {
		s.NotFound(w, r)
		return
	}

	artists := s.store.GetArtists()
	stats := s.store.GetStats()
	locations := s.store.GetLocations()

	data := struct {
		Title          string
		ExtraCSS       string
		ExtraJS        string
		Artists        []app.Artist
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

	s.render(w, r, "home.tmpl", data)
}

// Artists handles the artists listing page.
func (s *Server) Artists(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if r.URL.Path != "/artists" {
		s.NotFound(w, r)
		return
	}

	artists := s.store.GetArtists()

	data := struct {
		Title    string
		ExtraCSS string
		ExtraJS  string
		Artists  []app.Artist
	}{
		Title:    "Artists",
		ExtraCSS: "artists.css",
		ExtraJS:  "",
		Artists:  artists,
	}

	s.render(w, r, "artists.tmpl", data)
}

// ArtistDetail handles individual artist pages.
func (s *Server) ArtistDetail(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract artist identifier from URL
	path := strings.TrimPrefix(r.URL.Path, "/artists/")
	if path == "" {
		s.NotFound(w, r)
		return
	}

	var artist app.Artist
	var found bool

	// Try slug first, then ID
	if artist, found = s.store.GetArtistBySlug(path); !found {
		if id, err := strconv.Atoi(path); err == nil {
			artist, found = s.store.GetArtist(id)
		}
	}

	if !found {
		s.NotFound(w, r)
		return
	}

	concert, _ := s.store.GetConcert(artist.ID)
	prev, next := s.store.GetNextPrevArtist(artist)

	data := struct {
		Title      string
		ExtraCSS   string
		ExtraJS    string
		Artist     app.Artist
		Relation   app.Concert
		TotalShows int
		Countries  []string
		PrevArtist *app.Artist
		NextArtist *app.Artist
	}{
		Title:      artist.Name,
		ExtraCSS:   "artist_detail.css",
		ExtraJS:    "",
		Artist:     artist,
		Relation:   concert, // Using "Relation" for template compatibility
		TotalShows: s.store.CountShows(concert),
		Countries:  s.store.GetCountries(concert),
		PrevArtist: prev,
		NextArtist: next,
	}

	s.render(w, r, "artist_detail.tmpl", data)
}

// Locations handles the locations listing page.
func (s *Server) Locations(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if r.URL.Path != "/locations" {
		s.NotFound(w, r)
		return
	}

	locations := s.store.GetLocations()
	locationStats := s.store.GetLocationStats()
	globalStats := s.store.GetStats()

	data := struct {
		Title          string
		ExtraCSS       string
		ExtraJS        string
		Locations      []string
		LocationStats  []app.LocationStats
		TopLocations   []app.LocationStats
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

	s.render(w, r, "locations.tmpl", data)
}

// LocationDetail handles individual location pages.
func (s *Server) LocationDetail(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract location slug from URL
	slug := strings.TrimPrefix(r.URL.Path, "/locations/")
	if slug == "" {
		s.NotFound(w, r)
		return
	}

	location, found := s.store.GetLocationBySlug(slug)
	if !found {
		s.NotFound(w, r)
		return
	}

	data := struct {
		Title        string
		ExtraCSS     string
		ExtraJS      string
		LocationName string
		DisplayName  string
		Location     app.LocationStats
		Artists      []app.Artist
	}{
		Title:        fmt.Sprintf("%s - Location", location.DisplayName),
		ExtraCSS:     "locations.css",
		ExtraJS:      "",
		LocationName: location.Name,
		DisplayName:  location.DisplayName,
		Location:     location,
		Artists:      location.Artists,
	}

	s.render(w, r, "location_detail.tmpl", data)
}

// Health provides a health check endpoint.
func (s *Server) Health(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	response := map[string]interface{}{
		"status":    "healthy",
		"timestamp": r.Header.Get("X-Request-Time"),
		"stats":     s.store.GetStats(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// NotFound handles 404 errors.
func (s *Server) NotFound(w http.ResponseWriter, r *http.Request) {
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
	if err := s.templates.ExecuteTemplate(w, "error.tmpl", data); err != nil {
		log.Printf("Template execution error for error.tmpl: %v", err)
		// Don't try to call render again, just log the error
	}
}

// InternalError handles 500 errors.
func (s *Server) InternalError(w http.ResponseWriter, r *http.Request, message string) {
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
	if err := s.templates.ExecuteTemplate(w, "error.tmpl", data); err != nil {
		log.Printf("Template execution error for error.tmpl: %v", err)
		// Don't try to call render again, just log the error
	}
}

// DevPanic is a development endpoint to test panic recovery.
func (s *Server) DevPanic(w http.ResponseWriter, r *http.Request) {
	panic("Development panic triggered")
}

// Private helper methods

func (s *Server) loadTemplates() {
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
	s.templates, err = template.New("").Funcs(funcMap).ParseFiles(templateFiles...)
	if err != nil {
		log.Fatalf("Failed to parse templates: %v", err)
	}
}

func (s *Server) render(w http.ResponseWriter, r *http.Request, templateName string, data interface{}) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := s.templates.ExecuteTemplate(w, templateName, data); err != nil {
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
