// Package handlers provides HTTP request handlers for the Groupie Tracker application.
package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"groupie-tracker/internal/api"
	"groupie-tracker/internal/models"
	"groupie-tracker/internal/storage"
)

// Handlers contains all HTTP handlers and their dependencies.
type Handlers struct {
	store     *storage.Store
	templates *template.Template
	apiClient *api.Client
}

// Response structures for API endpoints
type SearchResponse struct {
	Artists []models.Artist `json:"artists"`
	Query   string          `json:"query"`
	Total   int             `json:"total"`
}

type SuggestResponse struct {
	Suggestions []string `json:"suggestions"`
	Query       string   `json:"query"`
}

type HealthResponse struct {
	Status string         `json:"status"`
	Stats  map[string]int `json:"stats"`
}

// LocationStat represents statistics for a location
type LocationStat struct {
	Name         string
	ArtistCount  int
	ConcertCount int
	Artists      []models.Artist
}

// NewHandlers creates a new handlers instance with the given store and API client.
func NewHandlers(store *storage.Store) *Handlers {
	h := &Handlers{
		store: store,
	}
	h.loadTemplates()
	return h
}

// loadTemplates loads all HTML templates
func (h *Handlers) loadTemplates() {
	templateFiles := []string{
		"templates/base.tmpl",
		"templates/home.tmpl",
		"templates/artists.tmpl",
		"templates/artist_detail.tmpl",
		"templates/locations.tmpl",
		"templates/error.tmpl",
	}

	// Define custom template functions
	funcMap := template.FuncMap{
		"sub": func(a, b int) int {
			return a - b
		},
		"add": func(a, b int) int {
			return a + b
		},
		"contains": func(s, substr string) bool {
			return strings.Contains(s, substr)
		},
	}

	var err error
	h.templates, err = template.New("").Funcs(funcMap).ParseFiles(templateFiles...)
	if err != nil {
		log.Printf("Warning: Could not load templates: %v", err)
		// Create a simple fallback template
		h.templates = template.Must(template.New("fallback").Parse(`
			<!DOCTYPE html>
			<html><head><title>{{.Title}}</title></head>
			<body><h1>{{.Title}}</h1><div>{{.Content}}</div></body></html>
		`))
	}
}

// SetAPIClient sets the API client for the handlers.
func (h *Handlers) SetAPIClient(client *api.Client) {
	h.apiClient = client
}

// SetTemplates sets the template instance for rendering HTML pages.
func (h *Handlers) SetTemplates(tmpl *template.Template) {
	h.templates = tmpl
}

// HomeHandler handles the home page.
func (h *Handlers) HomeHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	artists := h.store.GetAllArtists()

	data := struct {
		Title          string
		Artists        []models.Artist
		ExtraCSS       string
		ExtraJS        string
		TotalMembers   int
		TotalLocations int
	}{
		Title:          "Home",
		Artists:        artists,
		ExtraCSS:       "home.css",
		ExtraJS:        "",
		TotalMembers:   calculateTotalMembers(artists),
		TotalLocations: len(h.store.GetUniqueLocations()),
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	// Execute the base template which will include the home content
	if h.templates != nil {
		if err := h.templates.ExecuteTemplate(w, "base.tmpl", data); err != nil {
			log.Printf("Template execution error: %v", err)
			// Fallback to simple HTML
			h.writeSimpleHTML(w, "Home", fmt.Sprintf("Found %d artists", len(artists)))
		}
	} else {
		h.writeSimpleHTML(w, "Home", fmt.Sprintf("Found %d artists", len(artists)))
	}
}

// writeSimpleHTML writes a simple HTML response as fallback
func (h *Handlers) writeSimpleHTML(w http.ResponseWriter, title, content string) {
	html := fmt.Sprintf(`
		<!DOCTYPE html>
		<html>
		<head><title>%s - Groupie Tracker</title></head>
		<body><h1>%s</h1><p>%s</p></body>
		</html>
	`, title, title, content)
	w.Write([]byte(html))
}

// ArtistsHandler handles the artists listing page.
func (h *Handlers) ArtistsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	artists := h.store.GetAllArtists()

	data := struct {
		Title    string
		Artists  []models.Artist
		ExtraCSS string
		ExtraJS  string
	}{
		Title:    "Artists",
		Artists:  artists,
		ExtraCSS: "artists.css",
		ExtraJS:  "",
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	// Execute the base template which will include the artists content
	if h.templates != nil {
		if err := h.templates.ExecuteTemplate(w, "base.tmpl", data); err != nil {
			log.Printf("Template execution error: %v", err)
			// Fallback to simple HTML
			h.writeSimpleHTML(w, "Artists", fmt.Sprintf("Found %d artists", len(artists)))
		}
	} else {
		h.writeSimpleHTML(w, "Artists", fmt.Sprintf("Found %d artists", len(artists)))
	}
}

// ArtistDetailHandler handles individual artist detail pages.
func (h *Handlers) ArtistDetailHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract artist ID from URL path
	path := strings.TrimPrefix(r.URL.Path, "/artists/")
	if path == "" {
		http.Redirect(w, r, "/artists", http.StatusSeeOther)
		return
	}

	id, err := strconv.Atoi(path)
	if err != nil {
		http.Error(w, "Invalid artist ID", http.StatusBadRequest)
		return
	}

	artist, exists := h.store.GetArtist(id)
	if !exists {
		h.NotFoundHandler(w, r)
		return
	}

	// Get relations data for this artist
	relations, _ := h.store.GetRelation(id)

	// Calculate total concerts
	totalConcerts := 0
	for _, dates := range relations.DatesLocations {
		totalConcerts += len(dates)
	}

	// Get previous and next artists for navigation
	allArtists := h.store.GetAllArtists()
	var prevArtist, nextArtist *models.Artist
	for i, a := range allArtists {
		if a.ID == id {
			if i > 0 {
				prevArtist = &allArtists[i-1]
			}
			if i < len(allArtists)-1 {
				nextArtist = &allArtists[i+1]
			}
			break
		}
	}

	data := struct {
		Title         string
		Artist        models.Artist
		Relations     *models.Relation
		TotalConcerts int
		PrevArtist    *models.Artist
		NextArtist    *models.Artist
		ExtraCSS      string
		ExtraJS       string
	}{
		Title:         artist.Name,
		Artist:        artist,
		Relations:     &relations,
		TotalConcerts: totalConcerts,
		PrevArtist:    prevArtist,
		NextArtist:    nextArtist,
		ExtraCSS:      "artist_detail.css",
		ExtraJS:       "",
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	// Execute the base template which will include the artist detail content
	if h.templates != nil {
		if err := h.templates.ExecuteTemplate(w, "base.tmpl", data); err != nil {
			log.Printf("Template execution error: %v", err)
			// Fallback to simple HTML
			h.writeSimpleHTML(w, artist.Name, fmt.Sprintf("Artist: %s (%d)", artist.Name, artist.CreationYear))
		}
	} else {
		h.writeSimpleHTML(w, artist.Name, fmt.Sprintf("Artist: %s (%d)", artist.Name, artist.CreationYear))
	}
}

// SearchHandler handles search API requests.
func (h *Handlers) SearchHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	query := r.URL.Query().Get("q")

	var artists []models.Artist
	if query == "" {
		artists = h.store.GetAllArtists()
	} else {
		artists = h.store.SearchArtists(query)
	}

	response := SearchResponse{
		Artists: artists,
		Query:   query,
		Total:   len(artists),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(response); err != nil {
		h.InternalErrorHandler(w, r, "Failed to encode response")
		return
	}
}

// SuggestHandler handles autocomplete suggestion requests.
func (h *Handlers) SuggestHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	query := r.URL.Query().Get("q")

	var suggestions []string
	if query != "" && len(query) >= 2 {
		artists := h.store.SearchArtists(query)

		// Create suggestions based on matching artists
		suggestionMap := make(map[string]bool)

		for _, artist := range artists {
			// Add artist name if it matches
			if strings.Contains(strings.ToLower(artist.Name), strings.ToLower(query)) {
				suggestionMap[artist.Name] = true
			}

			// Add member names if they match
			for _, member := range artist.Members {
				if strings.Contains(strings.ToLower(member), strings.ToLower(query)) {
					suggestionMap[member] = true
				}
			}
		}

		// Convert map to slice
		for suggestion := range suggestionMap {
			suggestions = append(suggestions, suggestion)
		}
	}

	response := SuggestResponse{
		Suggestions: suggestions,
		Query:       query,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(response); err != nil {
		h.InternalErrorHandler(w, r, "Failed to encode response")
		return
	}
}

// HealthHandler handles health check requests.
func (h *Handlers) HealthHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	stats := h.store.GetStats()

	response := HealthResponse{
		Status: "healthy",
		Stats:  stats,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Failed to encode health response: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
}

// NotFoundHandler handles 404 errors.
func (h *Handlers) NotFoundHandler(w http.ResponseWriter, r *http.Request) {
	data := struct {
		Title        string
		ErrorCode    int
		ErrorMessage string
		RequestedURL string
		Timestamp    string
		ExtraCSS     string
		ExtraJS      string
	}{
		Title:        "Page Not Found",
		ErrorCode:    404,
		ErrorMessage: "The page you're looking for doesn't exist or has been moved.",
		RequestedURL: r.URL.Path,
		Timestamp:    time.Now().Format("2006-01-02 15:04:05"),
		ExtraCSS:     "errors.css",
		ExtraJS:      "",
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusNotFound)

	// Execute the base template which will include the error content
	if h.templates != nil {
		if err := h.templates.ExecuteTemplate(w, "base.tmpl", data); err != nil {
			log.Printf("Template execution error: %v", err)
			// Fallback to simple HTML
			h.writeSimpleHTML(w, "Page Not Found", "The page you requested could not be found.")
		}
	} else {
		h.writeSimpleHTML(w, "Page Not Found", "The page you requested could not be found.")
	}
}

// InternalErrorHandler handles 500 errors.
func (h *Handlers) InternalErrorHandler(w http.ResponseWriter, r *http.Request, message string) {
	log.Printf("Internal server error: %s", message)

	data := struct {
		Title        string
		ErrorCode    int
		ErrorMessage string
		RequestedURL string
		Timestamp    string
		ExtraCSS     string
		ExtraJS      string
	}{
		Title:        "Internal Server Error",
		ErrorCode:    500,
		ErrorMessage: message,
		RequestedURL: r.URL.Path,
		Timestamp:    time.Now().Format("2006-01-02 15:04:05"),
		ExtraCSS:     "errors.css",
		ExtraJS:      "",
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusInternalServerError)

	// Execute the base template which will include the error content
	if h.templates != nil {
		if err := h.templates.ExecuteTemplate(w, "base.tmpl", data); err != nil {
			log.Printf("Template execution error: %v", err)
			// Fallback to simple HTML
			h.writeSimpleHTML(w, "Internal Server Error", "Something went wrong on our end. Please try again later.")
		}
	} else {
		h.writeSimpleHTML(w, "Internal Server Error", "Something went wrong on our end. Please try again later.")
	}
}

// LocationsHandler handles the locations page.
func (h *Handlers) LocationsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	locations := h.store.GetUniqueLocations()
	locationStats := h.calculateLocationStats()

	data := struct {
		Title          string
		Locations      []string
		LocationStats  []LocationStat
		TopLocations   []LocationStat
		TotalCountries int
		TotalConcerts  int
		ExtraCSS       string
		ExtraJS        string
	}{
		Title:          "Locations",
		Locations:      locations,
		LocationStats:  locationStats,
		TopLocations:   locationStats, // Same data, template will limit to top 10
		TotalCountries: h.calculateTotalCountries(locationStats),
		TotalConcerts:  h.calculateTotalConcerts(),
		ExtraCSS:       "locations.css",
		ExtraJS:        "",
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	// Execute the base template which will include the locations content
	if h.templates != nil {
		if err := h.templates.ExecuteTemplate(w, "base.tmpl", data); err != nil {
			log.Printf("Template execution error: %v", err)
			// Fallback to simple HTML
			h.writeSimpleHTML(w, "Locations", fmt.Sprintf("Found %d locations", len(locations)))
		}
	} else {
		h.writeSimpleHTML(w, "Locations", fmt.Sprintf("Found %d locations", len(locations)))
	}
}

// RefreshHandler handles data refresh requests (POST /api/refresh).
func (h *Handlers) RefreshHandler(w http.ResponseWriter, r *http.Request) {
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

	// Fetch fresh data from API
	data, err := h.apiClient.FetchAllData(ctx)
	if err != nil {
		log.Printf("Failed to refresh data: %v", err)
		http.Error(w, "Failed to refresh data", http.StatusInternalServerError)
		return
	}

	// Update store with new data (APIResponse)
	h.store.LoadData(*data)

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

	log.Printf("Data refreshed: %d artists, %d locations, %d dates, %d relations",
		len(data.Artists), len(data.Locations), len(data.Dates), len(data.Relations))
}

// calculateTotalMembers calculates the total number of members across all artists
func calculateTotalMembers(artists []models.Artist) int {
	total := 0
	for _, artist := range artists {
		total += len(artist.Members)
	}
	return total
}

// calculateLocationStats calculates statistics for each location
func (h *Handlers) calculateLocationStats() []LocationStat {
	locationMap := make(map[string]*LocationStat)
	allArtists := h.store.GetAllArtists()
	allRelations := h.store.GetAllRelations()

	// Create a map of artist ID to artist for quick lookup
	artistMap := make(map[int]models.Artist)
	for _, artist := range allArtists {
		artistMap[artist.ID] = artist
	}

	// Process each relation to build location statistics
	for _, relation := range allRelations {
		artist, exists := artistMap[relation.ID]
		if !exists {
			continue
		}

		for location, dates := range relation.DatesLocations {
			if locationMap[location] == nil {
				locationMap[location] = &LocationStat{
					Name:         location,
					ArtistCount:  0,
					ConcertCount: 0,
					Artists:      []models.Artist{},
				}
			}

			locationMap[location].ArtistCount++
			locationMap[location].ConcertCount += len(dates)
			locationMap[location].Artists = append(locationMap[location].Artists, artist)
		}
	}

	// Convert map to slice
	var locationStats []LocationStat
	for _, stat := range locationMap {
		locationStats = append(locationStats, *stat)
	}

	return locationStats
}

// calculateTotalCountries calculates the total number of unique countries
func (h *Handlers) calculateTotalCountries(locationStats []LocationStat) int {
	countrySet := make(map[string]bool)

	for _, stat := range locationStats {
		// Extract country from location (assuming format "city-country")
		parts := strings.Split(stat.Name, "-")
		if len(parts) >= 2 {
			country := strings.TrimSpace(parts[len(parts)-1])
			countrySet[country] = true
		}
	}

	return len(countrySet)
}

// calculateTotalConcerts calculates the total number of concerts across all artists
func (h *Handlers) calculateTotalConcerts() int {
	total := 0
	allRelations := h.store.GetAllRelations()

	for _, relation := range allRelations {
		for _, dates := range relation.DatesLocations {
			total += len(dates)
		}
	}

	return total
}
