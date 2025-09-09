// Package handlers provides HTTP request handlers using simplified architecture.
package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"time"

	"groupie-tracker/internal/api"
	"groupie-tracker/internal/models"
	"groupie-tracker/internal/service"
	"groupie-tracker/internal/storage"
)

// SimplifiedHandlers contains all HTTP handlers using the simplified architecture.
type SimplifiedHandlers struct {
	store     *storage.SimplifiedStore
	service   *service.SimplifiedService
	templates *template.Template
	apiClient *api.Client
}

// SimplifiedLocationStat uses the service's LocationStat for consistency
type SimplifiedLocationStat = service.LocationStat

// NewSimplifiedHandlers creates a new handlers instance with simplified architecture.
func NewSimplifiedHandlers(store *storage.SimplifiedStore, apiClient *api.Client) *SimplifiedHandlers {
	// Create service that uses the store
	svc := service.NewSimplifiedService(store)

	h := &SimplifiedHandlers{
		store:     store,
		service:   svc,
		apiClient: apiClient,
	}
	h.loadTemplates()
	return h
}

// loadTemplates loads all HTML templates
func (h *SimplifiedHandlers) loadTemplates() {
	templateFiles := []string{
		"templates/base.tmpl",
		"templates/home.tmpl",
		"templates/artists.tmpl",
		"templates/artist_detail.tmpl",
		"templates/locations.tmpl",
		"templates/error.tmpl",
	}

	funcMap := template.FuncMap{
		"add": func(a, b int) int { return a + b },
		"sub": func(a, b int) int { return a - b },
		"contains": func(slice []string, item string) bool {
			for _, s := range slice {
				if s == item {
					return true
				}
			}
			return false
		},
		"safeLen": func(slice interface{}) int {
			if slice == nil {
				return 0
			}
			switch s := slice.(type) {
			case []string:
				return len(s)
			case []models.Artist:
				return len(s)
			default:
				return 0
			}
		},
	}

	var err error
	h.templates, err = template.New("").Funcs(funcMap).ParseFiles(templateFiles...)
	if err != nil {
		log.Printf("Warning: Could not load templates: %v", err)
		h.templates = nil
	}
}

// LocationsHandler handles the locations page using simplified architecture.
func (h *SimplifiedHandlers) LocationsHandler(w http.ResponseWriter, r *http.Request) {
	defer func() {
		if err := recover(); err != nil {
			log.Printf("Panic recovered in LocationsHandler: %v", err)
			h.InternalErrorHandler(w, r, fmt.Sprintf("Panic: %v", err))
		}
	}()

	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Use simplified service for business logic
	locations := h.store.GetUniqueLocations()
	locationStats := h.service.CalculateLocationStats()
	sortedLocationStats := h.service.SortLocationStatsByConcertCount(locationStats)

	data := struct {
		Title          string
		Locations      []string
		LocationStats  []SimplifiedLocationStat
		TopLocations   []SimplifiedLocationStat
		TotalCountries int
		TotalConcerts  int
		ExtraCSS       string
		ExtraJS        string
	}{
		Title:          "Locations",
		Locations:      locations,
		LocationStats:  locationStats,
		TopLocations:   sortedLocationStats, // Now properly sorted!
		TotalCountries: h.service.CalculateTotalCountries(locationStats),
		TotalConcerts:  h.service.CalculateTotalConcerts(),
		ExtraCSS:       "locations.css",
		ExtraJS:        "",
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	// Execute the locations template
	if h.templates != nil {
		if err := h.templates.ExecuteTemplate(w, "locations.tmpl", data); err != nil {
			log.Printf("Template execution error: %v", err)
			h.writeSimpleHTML(w, "Locations", fmt.Sprintf("Found %d locations", len(locations)))
		}
	} else {
		h.writeSimpleHTML(w, "Locations", fmt.Sprintf("Found %d locations", len(locations)))
	}
}

// HomeHandler handles the home page.
func (h *SimplifiedHandlers) HomeHandler(w http.ResponseWriter, r *http.Request) {
	defer func() {
		if err := recover(); err != nil {
			log.Printf("Panic recovered in HomeHandler: %v", err)
			h.InternalErrorHandler(w, r, fmt.Sprintf("Panic: %v", err))
		}
	}()

	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	artists := h.store.GetAllArtists()
	stats := h.store.GetStats()

	// Get featured artists (first 6 for display)
	featuredArtists := artists
	if len(featuredArtists) > 6 {
		featuredArtists = featuredArtists[:6]
	}

	data := struct {
		Title           string
		Artists         []models.Artist
		FeaturedArtists []models.Artist
		Stats           map[string]int
		ExtraCSS        string
		ExtraJS         string
	}{
		Title:           "Home",
		Artists:         artists,
		FeaturedArtists: featuredArtists,
		Stats:           stats,
		ExtraCSS:        "home.css",
		ExtraJS:         "",
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	if h.templates != nil {
		if err := h.templates.ExecuteTemplate(w, "home.tmpl", data); err != nil {
			log.Printf("Template execution error: %v", err)
			h.writeSimpleHTML(w, "Home", "Welcome to Groupie Tracker")
		}
	} else {
		h.writeSimpleHTML(w, "Home", "Welcome to Groupie Tracker")
	}
}

// InternalErrorHandler handles internal server errors.
func (h *SimplifiedHandlers) InternalErrorHandler(w http.ResponseWriter, r *http.Request, message string) {
	log.Printf("Internal error: %s", message)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusInternalServerError)

	if h.templates != nil {
		data := struct {
			Title   string
			Message string
			Code    int
		}{
			Title:   "Internal Server Error",
			Message: "Something went wrong on our end.",
			Code:    500,
		}

		if err := h.templates.ExecuteTemplate(w, "error.tmpl", data); err != nil {
			log.Printf("Error template execution failed: %v", err)
			h.writeSimpleHTML(w, "Error", "Internal Server Error")
		}
	} else {
		h.writeSimpleHTML(w, "Error", "Internal Server Error")
	}
}

// writeSimpleHTML writes a simple HTML response when templates are not available.
func (h *SimplifiedHandlers) writeSimpleHTML(w http.ResponseWriter, title, content string) {
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

// RefreshHandler handles data refresh requests using simplified architecture.
func (h *SimplifiedHandlers) RefreshHandler(w http.ResponseWriter, r *http.Request) {
	defer func() {
		if err := recover(); err != nil {
			log.Printf("Panic recovered in RefreshHandler: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
		}
	}()

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if h.apiClient == nil {
		http.Error(w, "API client not configured", http.StatusInternalServerError)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Use simplified store's refresh functionality
	err := h.store.RefreshData(ctx)
	if err != nil {
		log.Printf("Failed to refresh data: %v", err)
		http.Error(w, "Failed to refresh data", http.StatusInternalServerError)
		return
	}

	// Return success response
	response := struct {
		Status  string         `json:"status"`
		Message string         `json:"message"`
		Stats   map[string]int `json:"stats"`
	}{
		Status:  "success",
		Message: "Data refreshed successfully",
		Stats:   h.store.GetStats(),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(response); err != nil {
		h.InternalErrorHandler(w, r, "Failed to encode refresh response")
		return
	}

	stats := h.store.GetStats()
	log.Printf("Data refreshed: %d artists, %d locations, %d dates, %d relations",
		stats["artists"], stats["locations"], stats["dates"], stats["relations"])
}

// ArtistsHandler handles requests to /artists page
func (h *SimplifiedHandlers) ArtistsHandler(w http.ResponseWriter, r *http.Request) {
	defer func() {
		if err := recover(); err != nil {
			log.Printf("Panic recovered in ArtistsHandler: %v", err)
			h.InternalErrorHandler(w, r, fmt.Sprintf("Panic: %v", err))
		}
	}()

	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	artists := h.store.GetAllArtists()

	if h.templates == nil {
		h.writeSimpleHTML(w, "Artists", fmt.Sprintf("Found %d artists", len(artists)))
		return
	}

	data := map[string]interface{}{
		"Artists": artists,
		"Title":   "Artists - Groupie Tracker",
	}

	if err := h.templates.ExecuteTemplate(w, "artists.tmpl", data); err != nil {
		log.Printf("Template execution error: %v", err)
		h.writeSimpleHTML(w, "Artists", fmt.Sprintf("Found %d artists", len(artists)))
	}
}

// ArtistDetailHandler handles requests to /artists/{id} or /artists/{slug}
func (h *SimplifiedHandlers) ArtistDetailHandler(w http.ResponseWriter, r *http.Request) {
	defer func() {
		if err := recover(); err != nil {
			log.Printf("Panic recovered in ArtistDetailHandler: %v", err)
			h.InternalErrorHandler(w, r, fmt.Sprintf("Panic: %v", err))
		}
	}()

	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract artist identifier from URL path
	path := r.URL.Path
	if len(path) <= len("/artists/") {
		h.NotFoundHandler(w, r)
		return
	}

	identifier := path[len("/artists/"):]

	// Try to get artist by slug first, then by ID
	var artist *models.Artist

	// Check if it's a slug
	foundArtist, exists := h.store.GetArtistBySlug(identifier)
	if exists {
		artist = &foundArtist
	}

	if artist == nil {
		h.NotFoundHandler(w, r)
		return
	}

	if h.templates == nil {
		h.writeSimpleHTML(w, "Artist Detail", fmt.Sprintf("Artist: %s (%d)", artist.Name, artist.CreationYear))
		return
	}

	data := map[string]interface{}{
		"Artist": artist,
		"Title":  fmt.Sprintf("%s - Groupie Tracker", artist.Name),
	}

	if err := h.templates.ExecuteTemplate(w, "artist_detail.tmpl", data); err != nil {
		log.Printf("Template execution error: %v", err)
		h.writeSimpleHTML(w, "Artist Detail", fmt.Sprintf("Artist: %s", artist.Name))
	}
}

// SearchHandler handles API search requests
func (h *SimplifiedHandlers) SearchHandler(w http.ResponseWriter, r *http.Request) {
	defer func() {
		if err := recover(); err != nil {
			log.Printf("Panic recovered in SearchHandler: %v", err)
			h.InternalErrorHandler(w, r, fmt.Sprintf("Panic: %v", err))
		}
	}()

	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	query := r.URL.Query().Get("q")
	artists := h.store.SearchArtists(query)

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(artists); err != nil {
		h.InternalErrorHandler(w, r, fmt.Sprintf("Failed to encode response: %v", err))
	}
}

// SuggestHandler handles API autocomplete suggestions
func (h *SimplifiedHandlers) SuggestHandler(w http.ResponseWriter, r *http.Request) {
	defer func() {
		if err := recover(); err != nil {
			log.Printf("Panic recovered in SuggestHandler: %v", err)
			h.InternalErrorHandler(w, r, fmt.Sprintf("Panic: %v", err))
		}
	}()

	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	query := r.URL.Query().Get("q")
	if len(query) < 2 {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]string{})
		return
	}

	artists := h.store.SearchArtists(query)
	suggestions := make([]string, 0, len(artists))
	for _, artist := range artists {
		suggestions = append(suggestions, artist.Name)
		if len(suggestions) >= 5 { // Limit suggestions
			break
		}
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(suggestions); err != nil {
		h.InternalErrorHandler(w, r, fmt.Sprintf("Failed to encode response: %v", err))
	}
}

// HealthHandler handles health check requests
func (h *SimplifiedHandlers) HealthHandler(w http.ResponseWriter, r *http.Request) {
	defer func() {
		if err := recover(); err != nil {
			log.Printf("Panic recovered in HealthHandler: %v", err)
			h.InternalErrorHandler(w, r, fmt.Sprintf("Panic: %v", err))
		}
	}()

	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	artists := h.store.GetAllArtists()
	status := map[string]interface{}{
		"status":  "healthy",
		"artists": len(artists),
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(status); err != nil {
		h.InternalErrorHandler(w, r, fmt.Sprintf("Failed to encode response: %v", err))
	}
}

// NotFoundHandler handles 404 errors
func (h *SimplifiedHandlers) NotFoundHandler(w http.ResponseWriter, r *http.Request) {
	defer func() {
		if err := recover(); err != nil {
			log.Printf("Panic recovered in NotFoundHandler: %v", err)
			h.InternalErrorHandler(w, r, fmt.Sprintf("Panic: %v", err))
		}
	}()

	w.WriteHeader(http.StatusNotFound)

	if h.templates == nil {
		h.writeSimpleHTML(w, "Page Not Found", "The requested page was not found.")
		return
	}

	data := map[string]interface{}{
		"Title":   "Page Not Found - Groupie Tracker",
		"Message": "The page you are looking for does not exist.",
		"Code":    404,
	}

	if err := h.templates.ExecuteTemplate(w, "error.tmpl", data); err != nil {
		log.Printf("Template execution error: %v", err)
		h.writeSimpleHTML(w, "Page Not Found", "The requested page was not found.")
	}
}
