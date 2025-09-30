package http

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"strings"

	"groupie-tracker/internal/models"
	"groupie-tracker/internal/service"
	"groupie-tracker/internal/store"
)

// Handler contains all HTTP handlers with dependencies injected.
type Handler struct {
	store         *store.DataStore
	templates     *TemplateRenderer
	searchService *service.SearchService
	filterService *service.FilterService
}

// NewHandler creates a new handler instance with all dependencies.
func NewHandler(
	store *store.DataStore,
	templates *TemplateRenderer,
	searchService *service.SearchService,
	filterService *service.FilterService,
) *Handler {
	return &Handler{
		store:         store,
		templates:     templates,
		searchService: searchService,
		filterService: filterService,
	}
}

// Routes sets up all HTTP routes with their respective handlers.
func (h *Handler) Routes() http.Handler {
	mux := http.NewServeMux()

	// Main application routes
	mux.HandleFunc("/", h.wrap(h.handleHome))
	mux.HandleFunc("/artists", h.wrap(h.handleArtists))
	mux.HandleFunc("/artists/", h.wrap(h.handleArtistDetail))
	mux.HandleFunc("/search", h.wrap(h.handleSearch))
	mux.HandleFunc("/locations", h.wrap(h.handleLocations))
	mux.HandleFunc("/locations/", h.wrap(h.handleLocationDetail))

	// API endpoints
	mux.HandleFunc("/api/suggestions", h.wrap(h.handleAPISuggestions))

	// Static file serving
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	return WithMiddleware(mux)
}

// wrap provides consistent error handling for all handlers.
func (h *Handler) wrap(fn func(w http.ResponseWriter, r *http.Request) error) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := fn(w, r); err != nil {
			h.handleError(w, r, err)
		}
	}
}

// handleError provides centralized error handling with proper logging and user feedback.
func (h *Handler) handleError(w http.ResponseWriter, r *http.Request, err error) {
	log.Printf("Handler error: %v (path: %s)", err, r.URL.Path)

	// Default to 500 if no specific status is provided
	status := http.StatusInternalServerError
	message := "Internal server error"

	// Check for specific error types to provide better responses
	if strings.Contains(err.Error(), "not found") {
		status = http.StatusNotFound
		message = "Page not found"
	} else if strings.Contains(err.Error(), "bad request") {
		status = http.StatusBadRequest
		message = "Bad request"
	}

	// Render error page
	data := struct {
		Title   string
		Status  int
		Message string
	}{
		Title:   "Error",
		Status:  status,
		Message: message,
	}

	w.WriteHeader(status)
	if err := h.templates.Render(w, "error.tmpl", data); err != nil {
		// Fallback to plain text if template rendering fails
		http.Error(w, message, status)
	}
}

// handleHome displays the home page with featured artists and search suggestions.
func (h *Handler) handleHome(w http.ResponseWriter, r *http.Request) error {
	if !h.validateExactPath(r, "/") {
		return fmt.Errorf("not found")
	}

	artists := h.store.GetAllArtists()
	suggestions := h.store.GetSuggestions()
	stats := h.store.GetStats()

	// Get random featured artists for the home page
	featuredArtists := h.getRandomArtists(artists, 8)

	data := struct {
		Title          string
		ExtraCSS       string
		Suggestions    []models.SearchSuggestion
		Artists        []models.Artist
		TotalArtists   int
		TotalLocations int
	}{
		Title:          "Home",
		ExtraCSS:       "home.css",
		Suggestions:    suggestions,
		Artists:        featuredArtists,
		TotalArtists:   stats.TotalArtists,
		TotalLocations: stats.TotalLocations,
	}

	return h.templates.Render(w, "home.tmpl", data)
}

// handleArtists displays the artists listing page with filtering capabilities.
func (h *Handler) handleArtists(w http.ResponseWriter, r *http.Request) error {
	if !h.validateExactPath(r, "/artists") {
		return fmt.Errorf("not found")
	}

	artists := h.store.GetAllArtists()
	var filters models.Filters
	var isFiltered bool

	// Handle POST request with filters
	if r.Method == http.MethodPost {
		if err := r.ParseForm(); err != nil {
			return fmt.Errorf("bad request: invalid form data")
		}

		filters = h.filterService.ParseFiltersFromForm(r.Form)
		if !filters.IsEmpty() {
			artists = h.filterService.FilterArtists(artists, filters)
			isFiltered = true
		}
	}

	// Compute filter options on-demand (cheap operation with small dataset)
	filterOptions := h.filterService.GetFilterOptions(h.store.GetAllArtists())

	data := struct {
		Title         string
		ExtraCSS      string
		Artists       []models.Artist
		Filters       models.Filters
		FilterOptions models.FilterOptions
		IsFiltered    bool
		ResultCount   int
	}{
		Title:         "Artists",
		ExtraCSS:      "artists.css",
		Artists:       artists,
		Filters:       filters,
		FilterOptions: filterOptions,
		IsFiltered:    isFiltered,
		ResultCount:   len(artists),
	}

	return h.templates.Render(w, "artists.tmpl", data)
}

// handleArtistDetail displays detailed information about a specific artist.
func (h *Handler) handleArtistDetail(w http.ResponseWriter, r *http.Request) error {
	// Extract slug from URL path
	slug := strings.TrimPrefix(r.URL.Path, "/artists/")
	if slug == "" {
		return fmt.Errorf("not found")
	}

	artist, exists := h.store.GetArtistBySlug(slug)
	if !exists {
		return fmt.Errorf("not found")
	}

	data := struct {
		Title    string
		ExtraCSS string
		Artist   models.Artist
	}{
		Title:    artist.Name,
		ExtraCSS: "artist_detail.css",
		Artist:   artist,
	}

	return h.templates.Render(w, "artist_detail.tmpl", data)
}

// handleSearch processes search queries and displays results.
func (h *Handler) handleSearch(w http.ResponseWriter, r *http.Request) error {
	if !h.validateExactPath(r, "/search") {
		return fmt.Errorf("not found")
	}

	// Parse search parameters
	searchParams := models.SearchParams{
		Query: strings.TrimSpace(r.URL.Query().Get("q")),
	}

	// Handle POST request with filters
	if r.Method == http.MethodPost {
		if err := r.ParseForm(); err != nil {
			return fmt.Errorf("bad request: invalid form data")
		}
		searchParams.Filters = h.filterService.ParseFiltersFromForm(r.Form)
	}

	// Perform search
	artists := h.store.GetAllArtists()
	searchResult := h.searchService.Search(artists, searchParams)

	// Get filter options for the search form
	filterOptions := h.filterService.GetFilterOptions(artists)

	data := struct {
		Title         string
		ExtraCSS      string
		Query         string
		Artists       []models.Artist
		TotalResults  int
		Filters       models.Filters
		FilterOptions models.FilterOptions
	}{
		Title:         "Search Results",
		ExtraCSS:      "search.css",
		Query:         searchParams.Query,
		Artists:       searchResult.Artists,
		TotalResults:  searchResult.TotalResults,
		Filters:       searchParams.Filters,
		FilterOptions: filterOptions,
	}

	return h.templates.Render(w, "search.tmpl", data)
}

// handleLocations displays the locations listing page.
func (h *Handler) handleLocations(w http.ResponseWriter, r *http.Request) error {
	if !h.validateExactPath(r, "/locations") {
		return fmt.Errorf("not found")
	}

	locations := h.store.GetAllLocations()

	data := struct {
		Title     string
		ExtraCSS  string
		Locations []models.Location
	}{
		Title:     "Locations",
		ExtraCSS:  "locations.css",
		Locations: locations,
	}

	return h.templates.Render(w, "locations.tmpl", data)
}

// handleLocationDetail displays detailed information about a specific location.
func (h *Handler) handleLocationDetail(w http.ResponseWriter, r *http.Request) error {
	// Extract slug from URL path
	slug := strings.TrimPrefix(r.URL.Path, "/locations/")
	if slug == "" {
		return fmt.Errorf("not found")
	}

	location, exists := h.store.GetLocationBySlug(slug)
	if !exists {
		return fmt.Errorf("not found")
	}

	data := struct {
		Title    string
		ExtraCSS string
		Location models.Location
	}{
		Title:    location.Name,
		ExtraCSS: "location_detail.css",
		Location: location,
	}

	return h.templates.Render(w, "location_detail.tmpl", data)
}

// handleAPISuggestions provides JSON API for search suggestions.
func (h *Handler) handleAPISuggestions(w http.ResponseWriter, r *http.Request) error {
	query := strings.TrimSpace(r.URL.Query().Get("q"))

	suggestions := h.store.GetSuggestions()
	filteredSuggestions := h.searchService.FilterSuggestions(suggestions, query)

	// Limit results for performance
	if len(filteredSuggestions) > 20 {
		filteredSuggestions = filteredSuggestions[:20]
	}

	w.Header().Set("Content-Type", "application/json")
	return json.NewEncoder(w).Encode(filteredSuggestions)
}

// Utility methods

// validateExactPath ensures the request path matches exactly (prevents path traversal).
func (h *Handler) validateExactPath(r *http.Request, expectedPath string) bool {
	return r.URL.Path == expectedPath
}

// getRandomArtists returns a random subset of artists for variety.
func (h *Handler) getRandomArtists(artists []models.Artist, count int) []models.Artist {
	if len(artists) <= count {
		return artists
	}

	// Simple random sampling without replacement
	indices := rand.Perm(len(artists))
	result := make([]models.Artist, count)
	for i := 0; i < count; i++ {
		result[i] = artists[indices[i]]
	}

	return result
}
