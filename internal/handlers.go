package data

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Strategic caching variables for expensive operations
var (
	dataStore   *DataStore
	templates   map[string]*template.Template
	suggestions []SearchSuggestion
)

// --- Server Configuration ---

const (
	defaultPort    = ":8082"
	readTimeout    = 10 * time.Second
	writeTimeout   = 10 * time.Second
	idleTimeout    = 60 * time.Second
	maxRequestSize = 32 << 20 // 32 MB
)

// --- Initialization Functions ---

// InitializeServer loads data and initializes caches.
func InitializeServer() error {
	start := time.Now()

	// Load all data from API
	log.Println("Loading initial data...")
	loadCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var err error
	dataStore, err = LoadData(loadCtx)
	if err != nil {
		return fmt.Errorf("failed to load data: %w", err)
	}

	// Initialize strategic caches
	log.Println("Initializing caches...")
	suggestions = GenerateSearchSuggestions(dataStore.Artists)
	
	if err := loadTemplates(); err != nil {
		return fmt.Errorf("failed to load templates: %w", err)
	}

	log.Printf("Server initialized in %v - %d artists, %d locations, %d suggestions",
		time.Since(start), len(dataStore.Artists), len(dataStore.Locations), len(suggestions))

	return nil
}

// CreateServer creates an HTTP server with all routes and middleware.
func CreateServer() *http.Server {
	mux := http.NewServeMux()

	// Static file handling
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))
	mux.Handle("/favicon.ico", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "static/favicon.ico")
	}))

	// Main routes
	mux.HandleFunc("/", homeHandler)
	mux.HandleFunc("/artists", artistsHandler)
	mux.HandleFunc("/artists/", artistDetailHandler)
	mux.HandleFunc("/locations", locationsHandler)
	mux.HandleFunc("/locations/", locationDetailHandler)
	mux.HandleFunc("/search", searchHandler)
	mux.HandleFunc("/api/suggestions", suggestionsAPIHandler)
	mux.HandleFunc("/health", healthHandler)

	// Wrap with middleware
	handler := withMiddleware(mux)

	port := getPort()
	return &http.Server{
		Addr:         port,
		Handler:      handler,
		ReadTimeout:  readTimeout,
		WriteTimeout: writeTimeout,
		IdleTimeout:  idleTimeout,
	}
}

// --- HTTP Handlers ---

// homeHandler handles the home page.
func homeHandler(w http.ResponseWriter, r *http.Request) {
	if !validateExactPath(w, r, "/") {
		return
	}

	artists := getRandomArtists(dataStore.Artists, 8)

	data := struct {
		Title          string
		ExtraCSS       string
		Suggestions    []SearchSuggestion
		Artists        []Artist
		TotalArtists   int
		TotalLocations int
	}{
		Title:          "Home",
		ExtraCSS:       "home.css",
		Suggestions:    suggestions,
		Artists:        artists,
		TotalArtists:   dataStore.Stats.TotalArtists,
		TotalLocations: dataStore.Stats.TotalLocations,
	}

	renderTemplate(w, r, "home.tmpl", data)
}

// artistsHandler handles the artists listing page with filtering.
func artistsHandler(w http.ResponseWriter, r *http.Request) {
	if !validateExactPath(w, r, "/artists") {
		return
	}

	artists := dataStore.Artists
	var filters Filters
	var isFiltered bool

	// Handle POST request with filters
	if r.Method == http.MethodPost {
		if err := r.ParseForm(); err != nil {
			errorHandler(w, r, http.StatusBadRequest, "Invalid form data")
			return
		}

		filters = ParseFiltersFromForm(r.Form)
		if !filters.IsEmpty() {
			artists = FilterArtists(artists, filters)
			isFiltered = true
		}
	}

	// Compute filter options on-demand (cheap operation)
	filterOptions := GetFilterOptions(dataStore.Artists)

	data := struct {
		Title         string
		ExtraCSS      string
		Suggestions   []SearchSuggestion
		Artists       []Artist
		FilterOptions FilterOptions
		AppliedFilters Filters
		IsFiltered    bool
		TotalArtists  int
	}{
		Title:          "Artists",
		ExtraCSS:       "artists.css",
		Suggestions:    suggestions,
		Artists:        artists,
		FilterOptions:  filterOptions,
		AppliedFilters: filters,
		IsFiltered:     isFiltered,
		TotalArtists:   len(dataStore.Artists),
	}

	renderTemplate(w, r, "artists.tmpl", data)
}

// artistDetailHandler handles individual artist pages.
func artistDetailHandler(w http.ResponseWriter, r *http.Request) {
	slug := strings.TrimPrefix(r.URL.Path, "/artists/")
	if slug == "" {
		notFoundHandler(w, r, "Artist not found")
		return
	}

	// Try slug first, then ID
	artist, exists := dataStore.ArtistsBySlug[slug]
	if !exists {
		if id, err := strconv.Atoi(slug); err == nil {
			artist, exists = dataStore.ArtistsByID[id]
		}
		if !exists {
			notFoundHandler(w, r, "Artist not found")
			return
		}
	}

	// Get navigation artists
	prevArtist, nextArtist := getAdjacentArtists(artist.ID)

	data := struct {
		Title      string
		ExtraCSS   string
		Suggestions []SearchSuggestion
		Artist     Artist
		PrevArtist *Artist
		NextArtist *Artist
	}{
		Title:       artist.Name,
		ExtraCSS:    "artist_detail.css",
		Suggestions: suggestions,
		Artist:      artist,
		PrevArtist:  prevArtist,
		NextArtist:  nextArtist,
	}

	renderTemplate(w, r, "artist_detail.tmpl", data)
}

// searchHandler handles search requests.
func searchHandler(w http.ResponseWriter, r *http.Request) {
	if !validateExactPath(w, r, "/search") {
		return
	}

	var params SearchParams
	var results SearchResult

	// Handle both GET and POST requests
	if r.Method == http.MethodPost {
		if err := r.ParseForm(); err != nil {
			errorHandler(w, r, http.StatusBadRequest, "Invalid form data")
			return
		}
		params.Query = strings.TrimSpace(r.FormValue("q"))
		params.Filters = ParseFiltersFromForm(r.Form)
	} else {
		params.Query = strings.TrimSpace(r.URL.Query().Get("q"))
	}

	// Perform search
	if params.Query != "" || !params.Filters.IsEmpty() {
		results = SearchArtists(dataStore.Artists, params)
	}

	// Compute filter options on-demand
	filterOptions := GetFilterOptions(dataStore.Artists)

	data := struct {
		Title         string
		ExtraCSS      string
		Suggestions   []SearchSuggestion
		Query         string
		Results       SearchResult
		FilterOptions FilterOptions
		Filters       Filters
	}{
		Title:         "Search",
		ExtraCSS:      "search.css",
		Suggestions:   suggestions,
		Query:         params.Query,
		Results:       results,
		FilterOptions: filterOptions,
		Filters:       params.Filters,
	}

	renderTemplate(w, r, "search.tmpl", data)
}

// locationsHandler handles the locations listing page.
func locationsHandler(w http.ResponseWriter, r *http.Request) {
	if !validateExactPath(w, r, "/locations") {
		return
	}

	data := struct {
		Title       string
		ExtraCSS    string
		Suggestions []SearchSuggestion
		Locations   []Location
	}{
		Title:       "Locations",
		ExtraCSS:    "locations.css",
		Suggestions: suggestions,
		Locations:   dataStore.Locations,
	}

	renderTemplate(w, r, "locations.tmpl", data)
}

// locationDetailHandler handles individual location pages.
func locationDetailHandler(w http.ResponseWriter, r *http.Request) {
	slug := strings.TrimPrefix(r.URL.Path, "/locations/")
	if slug == "" {
		notFoundHandler(w, r, "Location not found")
		return
	}

	location, exists := dataStore.LocationsBySlug[slug]
	if !exists {
		notFoundHandler(w, r, "Location not found")
		return
	}

	data := struct {
		Title       string
		ExtraCSS    string
		Suggestions []SearchSuggestion
		Location    Location
	}{
		Title:       location.Name,
		ExtraCSS:    "location_detail.css",
		Suggestions: suggestions,
		Location:    location,
	}

	renderTemplate(w, r, "location_detail.tmpl", data)
}

// --- API Handlers ---

// suggestionsAPIHandler provides JSON search suggestions.
func suggestionsAPIHandler(w http.ResponseWriter, r *http.Request) {
	query := strings.TrimSpace(r.URL.Query().Get("q"))
	
	filteredSuggestions := suggestions
	if query != "" {
		filteredSuggestions = FilterSuggestions(suggestions, query)
	}

	// Limit to first 20 suggestions for performance
	if len(filteredSuggestions) > 20 {
		filteredSuggestions = filteredSuggestions[:20]
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"suggestions": filteredSuggestions,
		"total":       len(filteredSuggestions),
	})
}

// healthHandler provides JSON health check.
func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "healthy",
		"stats":  dataStore.Stats,
	})
}

// --- Error Handlers ---

// notFoundHandler handles 404 errors.
func notFoundHandler(w http.ResponseWriter, r *http.Request, message string) {
	if message == "" {
		message = "The requested page was not found."
	}

	data := struct {
		Title        string
		ExtraCSS     string
		ErrorCode    int
		RequestedURL string
		Message      string
		Timestamp    string
	}{
		Title:        "Page Not Found",
		ExtraCSS:     "errors.css",
		ErrorCode:    404,
		RequestedURL: r.URL.Path,
		Message:      message,
		Timestamp:    time.Now().Format("2006-01-02 15:04:05"),
	}

	w.WriteHeader(http.StatusNotFound)
	renderTemplate(w, r, "error.tmpl", data)
}

// errorHandler handles general errors.
func errorHandler(w http.ResponseWriter, r *http.Request, code int, message string) {
	data := struct {
		Title        string
		ExtraCSS     string
		ErrorCode    int
		RequestedURL string
		Message      string
		Timestamp    string
	}{
		Title:        "Error",
		ExtraCSS:     "errors.css",
		ErrorCode:    code,
		RequestedURL: r.URL.Path,
		Message:      message,
		Timestamp:    time.Now().Format("2006-01-02 15:04:05"),
	}

	w.WriteHeader(code)
	renderTemplate(w, r, "error.tmpl", data)
}

// --- Utility Functions ---

// validateExactPath validates that the request path matches exactly.
func validateExactPath(w http.ResponseWriter, r *http.Request, expectedPath string) bool {
	if r.URL.Path != expectedPath {
		notFoundHandler(w, r, "")
		return false
	}
	return true
}

// getRandomArtists returns a random subset of artists.
func getRandomArtists(artists []Artist, count int) []Artist {
	if len(artists) <= count {
		return artists
	}

	// Simple deterministic selection for consistency
	step := len(artists) / count
	var selected []Artist
	for i := 0; i < count && i*step < len(artists); i++ {
		selected = append(selected, artists[i*step])
	}
	return selected
}

// getAdjacentArtists finds previous and next artists for navigation.
func getAdjacentArtists(artistID int) (*Artist, *Artist) {
	for i, artist := range dataStore.Artists {
		if artist.ID == artistID {
			var prev, next *Artist
			if i > 0 {
				prev = &dataStore.Artists[i-1]
			}
			if i < len(dataStore.Artists)-1 {
				next = &dataStore.Artists[i+1]
			}
			return prev, next
		}
	}
	return nil, nil
}

// getPort returns the port to listen on.
func getPort() string {
	if port := os.Getenv("PORT"); port != "" {
		return ":" + port
	}
	return defaultPort
}