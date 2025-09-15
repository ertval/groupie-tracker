// Package handlers provides HTTP request handlers for the Groupie Tracker application.
package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"time"

	"groupie-tracker/internal/api"
	"groupie-tracker/internal/models"
	"groupie-tracker/internal/service"
	"groupie-tracker/internal/storage"
)

// Handlers contains all HTTP handlers for the application.
type Handlers struct {
	store     *storage.Store
	service   *service.Service
	templates *template.Template
	apiClient *api.Client
}

// LocationStat uses the service's LocationStat for consistency
type LocationStat = service.LocationStat

// NewHandlers creates a new handlers instance.
func NewHandlers(store *storage.Store, service *service.Service, apiClient *api.Client) *Handlers {
	h := &Handlers{
		store:     store,
		service:   service,
		apiClient: apiClient,
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
		"templates/location_detail.tmpl",
		"templates/error.tmpl",
	}

	funcMap := template.FuncMap{
		"add": func(a, b int) int { return a + b },
		"sub": func(a, b int) int { return a - b },
		"contains": func(slice []string, item string) bool {
			return slices.Contains(slice, item)
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
		"join": func(items []string, sep string) string {
			return strings.Join(items, sep)
		},
		"generateLocationSlug": func(locationName string) string {
			return models.GenerateLocationSlug(locationName)
		},
		"normalizeLocationName": func(locationName string) string {
			return models.NormalizeLocationName(locationName)
		},
	}

	var err error
	h.templates, err = template.New("").Funcs(funcMap).ParseFiles(templateFiles...)
	if err != nil {
		log.Printf("Warning: Could not load templates: %v", err)
		h.templates = nil
	}
}

// HomeHandler handles the home page
func (h *Handlers) HomeHandler(w http.ResponseWriter, r *http.Request) {
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

	// Handle root path routing
	if r.URL.Path != "/" {
		h.NotFoundHandler(w, r)
		return
	}

	// Get data for the home page
	artists := h.service.GetAllArtistsSorted()
	stats := h.service.GetStats()
	locations := h.service.GetUniqueLocationsSorted()

	// Calculate total members
	totalMembers := 0
	for _, artist := range artists {
		totalMembers += len(artist.Members)
	}

	data := struct {
		Title          string
		Artists        []models.Artist
		Stats          map[string]int
		TotalArtists   int
		TotalMembers   int
		TotalLocations int
		ExtraCSS       string
		ExtraJS        string
	}{
		Title:          "Home",
		Artists:        artists,
		Stats:          stats,
		TotalArtists:   stats["artists"],
		TotalMembers:   totalMembers,
		TotalLocations: len(locations),
		ExtraCSS:       "home.css",
		ExtraJS:        "",
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

// LocationsHandler handles the locations page.
func (h *Handlers) LocationsHandler(w http.ResponseWriter, r *http.Request) {
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

	// Use service for business logic
	locations := h.service.GetUniqueLocationsSorted()
	locationStats := h.service.CalculateLocationStats() // This now returns sorted stats

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
		TopLocations:   locationStats, // Already sorted by CalculateLocationStats
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

// LocationDetailHandler handles requests to specific location pages
func (h *Handlers) LocationDetailHandler(w http.ResponseWriter, r *http.Request) {
	defer func() {
		if err := recover(); err != nil {
			log.Printf("Panic recovered in LocationDetailHandler: %v", err)
			h.InternalErrorHandler(w, r, fmt.Sprintf("Panic: %v", err))
		}
	}()

	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract location slug from URL path
	pathParts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(pathParts) < 2 {
		h.NotFoundHandler(w, r)
		return
	}

	locationSlug := pathParts[1]

	// Get location details from service
	locationDetail, found := h.service.GetLocationDetailsBySlug(locationSlug)
	if !found {
		h.NotFoundHandler(w, r)
		return
	}

	// Get concert dates for this location (unique across all artists)
	concertDates := h.service.GetLocationConcertDates(locationDetail.Name)

	// Get per-artist dates for this location
	artistsWithDates := h.service.GetArtistsWithDatesForLocation(locationDetail.Name)

	// Prepare data for template
	data := struct {
		Title            string
		LocationName     string
		DisplayName      string
		Artists          []models.Artist
		ArtistsWithDates []service.ArtistWithDates
		ConcertDates     []string
		ArtistCount      int
		ConcertCount     int
		ExtraCSS         string
		ExtraJS          string
	}{
		Title:            fmt.Sprintf("%s - Location", models.NormalizeLocationName(locationDetail.Name)),
		LocationName:     locationDetail.Name,
		DisplayName:      models.NormalizeLocationName(locationDetail.Name),
		Artists:          locationDetail.Artists,
		ArtistsWithDates: artistsWithDates,
		ConcertDates:     concertDates,
		ArtistCount:      locationDetail.ArtistCount,
		ConcertCount:     locationDetail.ConcertCount,
		ExtraCSS:         "locations.css",
		ExtraJS:          "",
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	// Execute the location detail template
	if h.templates != nil {
		if err := h.templates.ExecuteTemplate(w, "location_detail.tmpl", data); err != nil {
			log.Printf("Template execution error: %v", err)
			h.writeSimpleHTML(w, "Location Detail", fmt.Sprintf("Location: %s", data.DisplayName))
		}
	} else {
		h.writeSimpleHTML(w, "Location Detail", fmt.Sprintf("Location: %s", data.DisplayName))
	}
}

// ArtistsHandler handles requests to /artists page
func (h *Handlers) ArtistsHandler(w http.ResponseWriter, r *http.Request) {
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

	artists := h.service.GetAllArtistsSorted()
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

	if h.templates != nil {
		if err := h.templates.ExecuteTemplate(w, "artists.tmpl", data); err != nil {
			log.Printf("Template execution error: %v", err)
			h.writeSimpleHTML(w, "Artists", fmt.Sprintf("Found %d artists", len(artists)))
		}
	} else {
		h.writeSimpleHTML(w, "Artists", fmt.Sprintf("Found %d artists", len(artists)))
	}
}

// ArtistDetailHandler handles requests to specific artist pages
func (h *Handlers) ArtistDetailHandler(w http.ResponseWriter, r *http.Request) {
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
	pathParts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(pathParts) < 2 {
		h.NotFoundHandler(w, r)
		return
	}

	identifier := pathParts[1]
	var artist models.Artist
	var found bool

	// Try to get artist by slug first (SEO-friendly URLs)
	artist, found = h.store.GetArtistBySlug(identifier)
	if !found {
		// If slug doesn't work, try parsing as ID
		if id, err := strconv.Atoi(identifier); err == nil {
			artist, found = h.store.GetArtist(id)
		}
	}

	if !found {
		h.NotFoundHandler(w, r)
		return
	}

	// Get related data
	relation, _ := h.store.GetRelation(artist.ID)

	// Compute previous and next artist for navigation (based on alphabetical list)
	allArtists := h.service.GetAllArtistsSorted()
	prevArtist := (*models.Artist)(nil)
	nextArtist := (*models.Artist)(nil)
	currentIndex := -1
	for i, a := range allArtists {
		if a.ID == artist.ID {
			currentIndex = i
			break
		}
	}
	if currentIndex != -1 {
		if currentIndex > 0 {
			p := allArtists[currentIndex-1]
			prevArtist = &p
		}
		if currentIndex < len(allArtists)-1 {
			n := allArtists[currentIndex+1]
			nextArtist = &n
		}
	}

	data := struct {
		Title      string
		Artist     models.Artist
		Relation   models.Relation
		PrevArtist *models.Artist
		NextArtist *models.Artist
		TotalShows int
		Countries  []string
		ExtraCSS   string
		ExtraJS    string
	}{
		Title:      artist.Name,
		Artist:     artist,
		Relation:   relation,
		PrevArtist: prevArtist,
		NextArtist: nextArtist,
		TotalShows: h.service.CalculateTotalShows(relation),
		Countries:  h.service.ExtractCountries(relation),
		ExtraCSS:   "artist_detail.css",
		ExtraJS:    "",
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	if h.templates != nil {
		if err := h.templates.ExecuteTemplate(w, "artist_detail.tmpl", data); err != nil {
			log.Printf("Template execution error: %v", err)
			h.writeSimpleHTML(w, artist.Name, fmt.Sprintf("Artist: %s", artist.Name))
		}
	} else {
		h.writeSimpleHTML(w, artist.Name, fmt.Sprintf("Artist: %s", artist.Name))
	}
}

// SearchHandler handles search requests
func (h *Handlers) SearchHandler(w http.ResponseWriter, r *http.Request) {
	defer func() {
		if err := recover(); err != nil {
			log.Printf("Panic recovered in SearchHandler: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
		}
	}()

	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	query := r.URL.Query().Get("q")
	results := h.service.SearchArtists(query)

	response := struct {
		Query   string          `json:"query"`
		Results []models.Artist `json:"results"`
		Count   int             `json:"count"`
	}{
		Query:   query,
		Results: results,
		Count:   len(results),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(response); err != nil {
		h.InternalErrorHandler(w, r, "Failed to encode search response")
	}
}

// SuggestHandler handles autocomplete suggestions
func (h *Handlers) SuggestHandler(w http.ResponseWriter, r *http.Request) {
	defer func() {
		if err := recover(); err != nil {
			log.Printf("Panic recovered in SuggestHandler: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
		}
	}()

	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	query := r.URL.Query().Get("q")
	if len(query) < 2 {
		// Return empty suggestions for very short queries
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode([]string{})
		return
	}

	results := h.service.SearchArtists(query)
	suggestions := make([]string, 0, len(results))

	// Limit suggestions to first 10 results
	limit := 10
	if len(results) < limit {
		limit = len(results)
	}

	for i := 0; i < limit; i++ {
		suggestions = append(suggestions, results[i].Name)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(suggestions); err != nil {
		h.InternalErrorHandler(w, r, "Failed to encode suggestions response")
	}
}

// RefreshHandler handles data refresh requests.
func (h *Handlers) RefreshHandler(w http.ResponseWriter, r *http.Request) {
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

	// Use store's refresh functionality
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
		Stats:   h.service.GetStats(),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(response); err != nil {
		h.InternalErrorHandler(w, r, "Failed to encode refresh response")
		return
	}

	stats := h.service.GetStats()
	log.Printf("Data refreshed: %d artists, %d locations, %d dates, %d relations",
		stats["artists"], stats["locations"], stats["dates"], stats["relations"])
}

// HealthHandler handles health check requests
func (h *Handlers) HealthHandler(w http.ResponseWriter, r *http.Request) {
	defer func() {
		if err := recover(); err != nil {
			log.Printf("Panic recovered in HealthHandler: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
		}
	}()

	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	stats := h.service.GetStats()
	status := "ok"
	if stats["artists"] == 0 {
		status = "degraded"
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
		h.InternalErrorHandler(w, r, "Failed to encode health response")
	}
}

// NotFoundHandler handles 404 errors
func (h *Handlers) NotFoundHandler(w http.ResponseWriter, r *http.Request) {
	defer func() {
		if err := recover(); err != nil {
			log.Printf("Panic recovered in NotFoundHandler: %v", err)
			h.InternalErrorHandler(w, r, fmt.Sprintf("Panic: %v", err))
		}
	}()

	w.WriteHeader(http.StatusNotFound)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	if h.templates != nil {
		data := struct {
			Title        string
			Message      string
			ErrorCode    int
			RequestedURL string
			ExtraCSS     string
			ExtraJS      string
		}{
			Title:        "Page Not Found",
			Message:      "The page you're looking for doesn't exist.",
			ErrorCode:    404,
			RequestedURL: r.URL.Path,
			ExtraCSS:     "errors.css",
			ExtraJS:      "",
		}

		if err := h.templates.ExecuteTemplate(w, "error.tmpl", data); err != nil {
			log.Printf("Error template execution failed: %v", err)
			h.writeSimpleHTML(w, "Not Found", "Page not found")
		}
	} else {
		h.writeSimpleHTML(w, "Not Found", "Page not found")
	}
}

// InternalErrorHandler handles 500 errors
func (h *Handlers) InternalErrorHandler(w http.ResponseWriter, r *http.Request, message string) {
	log.Printf("Internal error: %s", message)

	w.WriteHeader(http.StatusInternalServerError)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	if h.templates != nil {
		data := struct {
			Title        string
			Message      string
			ErrorCode    int
			ErrorMessage string
			Timestamp    string
			ExtraCSS     string
			ExtraJS      string
		}{
			Title:        "Internal Server Error",
			Message:      "Something went wrong on our end.",
			ErrorCode:    500,
			ErrorMessage: message,
			Timestamp:    time.Now().Format("2006-01-02 15:04:05"),
			ExtraCSS:     "errors.css",
			ExtraJS:      "",
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

// PanicHandler is a dev/test handler that intentionally panics to exercise the recovery middleware.
// NOTE: This should only be used in development or test environments.
func (h *Handlers) PanicHandler(w http.ResponseWriter, r *http.Request) {
	panic("intentional panic triggered by PanicHandler")
}
