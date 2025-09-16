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
	"groupie-tracker/internal/data"
)

// Handlers contains all HTTP handlers for the application.
type Handlers struct {
	repo      *data.Repository
	templates *template.Template
	apiClient *api.Client
}

// PageData represents common data structure for all pages
type PageData struct {
	Title    string
	ExtraCSS string
	ExtraJS  string // Kept for template compatibility
}

// HomeData represents data needed for the home page
type HomeData struct {
	PageData
	Artists        []data.Artist
	Stats          map[string]int
	TotalArtists   int
	TotalMembers   int
	TotalLocations int
}

// ArtistsData represents data needed for the artists page
type ArtistsData struct {
	PageData
	Artists []data.Artist
}

// ArtistDetailData represents data needed for artist detail page
type ArtistDetailData struct {
	PageData
	Artist     data.Artist
	Relation   data.Relation
	PrevArtist *data.Artist
	NextArtist *data.Artist
	TotalShows int
	Countries  []string
}

// LocationsData represents data needed for locations page
type LocationsData struct {
	PageData
	Locations      []string
	LocationStats  []data.LocationStat
	TopLocations   []data.LocationStat // For template compatibility
	TotalCountries int
	TotalConcerts  int
}

// LocationDetailData represents data needed for location detail page
type LocationDetailData struct {
	PageData
	LocationName     string
	DisplayName      string
	Artists          []data.Artist
	ArtistsWithDates []data.ArtistWithDates
	ConcertDates     []string
	ArtistCount      int
	ConcertCount     int
}

// ErrorData represents data needed for error pages
type ErrorData struct {
	PageData
	Message      string
	ErrorCode    int
	RequestedURL string
	Timestamp    string
	ErrorMessage string
}

// NewHandlers creates a new handlers instance.
func NewHandlers(repo *data.Repository, apiClient *api.Client) *Handlers {
	h := &Handlers{
		repo:      repo,
		apiClient: apiClient,
	}
	h.loadTemplates()
	return h
}

// APIClientAdapter adapts the api.Client to work with the data.APIClient interface
type APIClientAdapter struct {
	Client *api.Client
}

// FetchAllData implements the data.APIClient interface
func (a *APIClientAdapter) FetchAllData(ctx context.Context) (*data.APIResponse, error) {
	apiResponse, err := a.Client.FetchAllData(ctx)
	if err != nil {
		return nil, err
	}

	// Convert api.APIResponse to data.APIResponse
	dataResponse := &data.APIResponse{
		Artists:   make([]data.APIArtist, len(apiResponse.Artists)),
		Relations: make([]data.APIRelation, len(apiResponse.Relations)),
	}

	// Convert artists
	for i, artist := range apiResponse.Artists {
		dataResponse.Artists[i] = data.APIArtist{
			ID:           artist.ID,
			Image:        artist.Image,
			Name:         artist.Name,
			Members:      artist.Members,
			CreationYear: artist.CreationYear,
			FirstAlbum:   artist.FirstAlbum,
		}
	}

	// Convert relations
	for i, relation := range apiResponse.Relations {
		dataResponse.Relations[i] = data.APIRelation{
			ID:             relation.ID,
			DatesLocations: relation.DatesLocations,
		}
	}

	return dataResponse, nil
}

// loadTemplates loads all HTML templates with helper functions
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

	// Verify critical templates exist
	if h.templates.Lookup("base.tmpl") == nil {
		log.Fatalf("base.tmpl template not found after parsing")
	}
	if h.templates.Lookup("home.tmpl") == nil {
		log.Fatalf("home.tmpl template not found after parsing")
	}
}

// HomeHandler handles the home page
func (h *Handlers) HomeHandler(w http.ResponseWriter, r *http.Request) {
	if !h.validateMethod(w, r, http.MethodGet) {
		return
	}

	// Handle root path routing
	if r.URL.Path != "/" {
		h.NotFoundHandler(w, r)
		return
	}

	// Get data for the home page
	artists := h.repo.GetAllArtistsSorted()
	stats := h.repo.GetStats()
	locations := h.repo.GetUniqueLocations()

	// Calculate total members via repository helper
	totalMembers := h.repo.GetTotalMembers()

	data := HomeData{
		PageData: PageData{
			Title:    "Home",
			ExtraCSS: "home.css",
			ExtraJS:  "", // Empty for now
		},
		Artists:        artists,
		Stats:          stats,
		TotalArtists:   stats["artists"],
		TotalMembers:   totalMembers,
		TotalLocations: len(locations),
	}

	h.executeTemplate(w, r, "home.tmpl", data)
}

// ArtistsHandler handles requests to /artists page
func (h *Handlers) ArtistsHandler(w http.ResponseWriter, r *http.Request) {
	if !h.validateMethod(w, r, http.MethodGet) {
		return
	}

	artists := h.repo.GetAllArtistsSorted()
	data := ArtistsData{
		PageData: PageData{
			Title:    "Artists",
			ExtraCSS: "artists.css",
			ExtraJS:  "", // Empty for now
		},
		Artists: artists,
	}

	h.executeTemplate(w, r, "artists.tmpl", data)
}

// ArtistDetailHandler handles requests to specific artist pages
func (h *Handlers) ArtistDetailHandler(w http.ResponseWriter, r *http.Request) {
	if !h.validateMethod(w, r, http.MethodGet) {
		return
	}

	// Extract artist identifier from URL path
	pathParts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(pathParts) != 2 {
		h.NotFoundHandler(w, r)
		return
	}

	identifier := pathParts[1]
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

	// Get related data
	relation, _ := h.repo.GetRelation(artist.ID)

	// Get previous and next artist for navigation
	prevArtist, nextArtist := h.repo.GetArtistNavigation(artist)

	data := ArtistDetailData{
		PageData: PageData{
			Title:    artist.Name,
			ExtraCSS: "artist_detail.css",
			ExtraJS:  "", // Empty for now
		},
		Artist:     artist,
		Relation:   relation,
		PrevArtist: prevArtist,
		NextArtist: nextArtist,
		TotalShows: h.repo.CalculateTotalShows(relation),
		Countries:  h.repo.ExtractCountries(relation),
	}

	h.executeTemplate(w, r, "artist_detail.tmpl", data)
}

// LocationsHandler handles the locations page.
func (h *Handlers) LocationsHandler(w http.ResponseWriter, r *http.Request) {
	if !h.validateMethod(w, r, http.MethodGet) {
		return
	}

	// Get data from repository
	locations := h.repo.GetUniqueLocations()
	locationStats := h.repo.CalculateLocationStats()

	// Calculate total countries via repository helper
	totalCountries := h.repo.GetTotalCountries()

	data := LocationsData{
		PageData: PageData{
			Title:    "Locations",
			ExtraCSS: "locations.css",
			ExtraJS:  "", // Empty for now
		},
		Locations:      locations,
		LocationStats:  locationStats, // Already sorted by CalculateLocationStats
		TopLocations:   locationStats, // Same as LocationStats for template compatibility
		TotalCountries: totalCountries,
		TotalConcerts:  h.repo.GetStats()["total_concerts"],
	}

	h.executeTemplate(w, r, "locations.tmpl", data)
}

// LocationDetailHandler handles requests to specific location pages
func (h *Handlers) LocationDetailHandler(w http.ResponseWriter, r *http.Request) {
	if !h.validateMethod(w, r, http.MethodGet) {
		return
	}

	// Extract location slug from URL path
	pathParts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(pathParts) != 2 {
		h.NotFoundHandler(w, r)
		return
	}

	locationSlug := pathParts[1]

	// Get location details from repository
	locationDetail, found := h.repo.GetLocationDetailsBySlug(locationSlug)
	if !found {
		h.NotFoundHandler(w, r)
		return
	}

	// Get per-artist dates for this location
	artistsWithDates := h.repo.GetArtistsWithDatesForLocation(locationDetail.Name)

	// Prepare data for template
	data := LocationDetailData{
		PageData: PageData{
			Title:    fmt.Sprintf("%s - Location", locationDetail.DisplayName),
			ExtraCSS: "locations.css",
			ExtraJS:  "", // Empty for now
		},
		LocationName:     locationDetail.Name,
		DisplayName:      locationDetail.DisplayName,
		Artists:          locationDetail.Artists,
		ArtistsWithDates: artistsWithDates,
		ConcertDates:     locationDetail.Dates,
		ArtistCount:      locationDetail.ArtistCount,
		ConcertCount:     locationDetail.ConcertCount,
	}

	h.executeTemplate(w, r, "location_detail.tmpl", data)
}

// HealthHandler handles health check requests
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

// NotFoundHandler handles 404 errors
func (h *Handlers) NotFoundHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotFound)

	data := ErrorData{
		PageData: PageData{
			Title:    "Page Not Found",
			ExtraCSS: "errors.css",
			ExtraJS:  "", // Empty for now
		},
		Message:      "The page you're looking for doesn't exist.",
		ErrorCode:    404,
		RequestedURL: r.URL.Path,
	}

	h.executeTemplate(w, r, "error.tmpl", data)
}

// InternalErrorHandler handles 500 errors
func (h *Handlers) InternalErrorHandler(w http.ResponseWriter, r *http.Request, message string) {
	log.Printf("Internal error: %s", message)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusInternalServerError)

	// Use direct template execution to avoid infinite recursion
	if h.templates != nil {
		data := ErrorData{
			PageData: PageData{
				Title:    "Internal Server Error",
				ExtraCSS: "errors.css",
				ExtraJS:  "", // Empty for now
			},
			Message:      "Something went wrong on our end. We're working to fix it!",
			ErrorCode:    500,
			ErrorMessage: message,
			Timestamp:    time.Now().Format("2006-01-02 15:04:05"),
		}

		if err := h.templates.ExecuteTemplate(w, "error.tmpl", data); err != nil {
			log.Printf("Template execution error: %v", err)
			h.writeSimpleHTML(w, "Internal Server Error", "An error occurred while rendering the page.")
		}
	} else {
		h.writeSimpleHTML(w, "Internal Server Error", "An error occurred and templates are not available.")
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

// validateMethod checks if the request method matches the expected method
func (h *Handlers) validateMethod(w http.ResponseWriter, r *http.Request, expectedMethod string) bool {
	if r.Method != expectedMethod {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return false
	}
	return true
}

// executeTemplate executes a template with error handling
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

// PanicHandler is a dev/test handler that intentionally panics to exercise the recovery middleware.
// NOTE: This should only be used in development or test environments.
func (h *Handlers) PanicHandler(w http.ResponseWriter, r *http.Request) {
	panic("This is an intentional panic for testing the recovery middleware")
}
