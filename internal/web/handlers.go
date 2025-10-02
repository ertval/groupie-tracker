package web

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"groupie-tracker/internal/data"
)

// ============================================================================
// STATIC FILES
// ============================================================================

// StaticFiles serves static assets like CSS, JS, images, and favicon.
func (app *App) StaticFiles(w http.ResponseWriter, r *http.Request) {
	const staticDir = "static"

	// Handle favicon.ico requests
	if r.URL.Path == "/favicon.ico" {
		target := filepath.Join(staticDir, "favicon.ico")
		if fi, err := os.Stat(target); err != nil || fi.IsDir() {
			app.NotFoundError(w, r, "Favicon not found")
			return
		}
		http.ServeFile(w, r, target)
		return
	}

	// Only allow /static/ prefix
	if !strings.HasPrefix(r.URL.Path, "/static/") {
		app.NotFoundError(w, r, "")
		return
	}

	// Extract relative path and prevent directory traversal
	rel := strings.TrimPrefix(r.URL.Path, "/static/")
	if rel == "" || strings.Contains(rel, "..") || strings.HasPrefix(rel, "/") {
		app.NotFoundError(w, r, "")
		return
	}

	// Build target path and verify it's a regular file
	target := filepath.Join(staticDir, rel)
	if fi, err := os.Stat(target); err != nil || fi.IsDir() {
		app.NotFoundError(w, r, "")
		return
	}

	// Serve the file (Go's http.ServeFile handles content-type automatically)
	http.ServeFile(w, r, target)
}

// ============================================================================
// HOME PAGE
// ============================================================================

// Home handles the home page.
func (app *App) Home(w http.ResponseWriter, r *http.Request) {
	// Validate path using centralized utility
	if !app.validateExactPath(w, r, "/") {
		return
	}

	artists := app.store.Artists()
	stats := app.store.Stats()
	suggestions := app.store.GenerateAllSearchSuggestions()

	// Get 8 random artists for homepage display
	artists = getRandomArtists(artists, 8)

	data := struct {
		Title          string
		ExtraCSS       string
		ExtraJS        string
		Suggestions    []data.SearchSuggestion
		Artists        []data.Artist
		TotalMembers   int
		TotalLocations int
	}{
		Title:          "Home",
		ExtraCSS:       "home.css",
		ExtraJS:        "",
		Suggestions:    suggestions,
		Artists:        artists,
		TotalMembers:   stats.TotalMembers,
		TotalLocations: stats.TotalLocations,
	}

	app.render(w, r, "home.tmpl", data)
}

// ============================================================================
// ARTISTS
// ============================================================================

// Artists handles the artists listing page.
func (app *App) Artists(w http.ResponseWriter, r *http.Request) {
	// Validate path for both GET and POST using centralized utility
	if !app.validateExactPath(w, r, "/artists") {
		return
	}

	artists := app.store.Artists()
	filterOptions := app.store.GetArtistFilterOptions()
	suggestions := app.store.GenerateAllSearchSuggestions()
	var appliedFilters data.ArtistFilterParams
	totalArtists := len(artists)

	// If POST request, parse form data and apply filters
	if r.Method == http.MethodPost {
		if !app.parseFormOrError(w, r) {
			return
		}

		appliedFilters = parseArtistFilterParams(r)
		artists = app.store.FilterArtists(appliedFilters)
	}

	// Sort artists by concert count (descending) for main display
	sort.Slice(artists, func(i, j int) bool {
		return artists[i].ConcertCount > artists[j].ConcertCount
	})

	data := struct {
		Title          string
		ExtraCSS       string
		ExtraJS        string
		Suggestions    []data.SearchSuggestion
		Artists        []data.Artist
		FilterOptions  data.ArtistFilterOptions
		AppliedFilters data.ArtistFilterParams
		IsFiltered     bool
		TotalArtists   int
	}{
		Title:          "Artists",
		ExtraCSS:       "artists.css",
		ExtraJS:        "",
		Suggestions:    suggestions,
		Artists:        artists,
		FilterOptions:  filterOptions,
		AppliedFilters: appliedFilters,
		IsFiltered:     r.Method == http.MethodPost,
		TotalArtists:   totalArtists,
	}

	app.render(w, r, "artists.tmpl", data)
}

// ArtistDetail handles individual artist pages.
func (app *App) ArtistDetail(w http.ResponseWriter, r *http.Request) {
	// Validate path
	path := strings.TrimPrefix(r.URL.Path, "/artists/")
	if path == "" {
		app.NotFoundError(w, r, "")
		return
	}

	// Try slug first, then ID
	artist, found := app.store.ArtistBySlug(path)
	if !found {
		if id, err := strconv.Atoi(path); err == nil {
			artist, found = app.store.ArtistByID(id)
		}
		if !found {
			app.NotFoundError(w, r, "Artist not found")
			return
		}
	}

	// Get navigation artists using on-demand lookup
	prevArtist, nextArtist := app.store.GetAdjacentArtists(artist.ID)
	suggestions := app.store.GenerateAllSearchSuggestions()

	data := struct {
		Title       string
		ExtraCSS    string
		ExtraJS     string
		Suggestions []data.SearchSuggestion
		Artist      data.Artist
		PrevArtist  *data.Artist
		NextArtist  *data.Artist
	}{
		Title:       artist.Name,
		ExtraCSS:    "artist_detail.css",
		ExtraJS:     "",
		Suggestions: suggestions,
		Artist:      artist,
		PrevArtist:  prevArtist,
		NextArtist:  nextArtist,
	}

	app.render(w, r, "artist_detail.tmpl", data)
}

// ============================================================================
// LOCATIONS
// ============================================================================

// Locations handles the locations listing page.
func (app *App) Locations(w http.ResponseWriter, r *http.Request) {
	// Validate path for both GET and POST using centralized utility
	if !app.validateExactPath(w, r, "/locations") {
		return
	}

	locations := app.store.Locations()
	filterOptions := app.store.GetLocationFilterOptions()
	suggestions := app.store.GenerateAllSearchSuggestions()
	var appliedFilters data.LocationFilterParams
	totalLocations := len(locations)
	stats := app.store.Stats()

	// If POST request, parse form data and apply filters
	if r.Method == http.MethodPost {
		if !app.parseFormOrError(w, r) {
			return
		}

		appliedFilters = parseLocationFilterParams(r)
		locations = app.store.FilterLocations(appliedFilters)
	}

	// Check if any filter is applied
	isFiltered := r.Method == http.MethodPost && (appliedFilters.ConcertCountFrom != nil || appliedFilters.ConcertCountTo != nil ||
		appliedFilters.ArtistCountFrom != nil || appliedFilters.ArtistCountTo != nil ||
		appliedFilters.ConcertYearFrom != nil || appliedFilters.ConcertYearTo != nil ||
		len(appliedFilters.Countries) > 0)

	// Generate filter description
	filterDescription := ""
	if isFiltered {
		if len(appliedFilters.Countries) > 0 {
			if len(appliedFilters.Countries) == 1 {
				filterDescription = appliedFilters.Countries[0]
			} else {
				filterDescription = "Multiple Countries"
			}
		} else {
			filterDescription = "Filters Applied"
		}
	}

	data := struct {
		Title                 string
		ExtraCSS              string
		ExtraJS               string
		Suggestions           []data.SearchSuggestion
		Locations             []data.Location
		LocationFilterOptions data.LocationFilterOptions
		AppliedFilters        data.LocationFilterParams
		IsFiltered            bool
		FilterDescription     string
		TotalLocations        int
		TotalCountries        int
		TotalConcerts         int
	}{
		Title:                 "Locations",
		ExtraCSS:              "locations.css",
		ExtraJS:               "",
		Suggestions:           suggestions,
		Locations:             locations,
		LocationFilterOptions: filterOptions,
		AppliedFilters:        appliedFilters,
		IsFiltered:            isFiltered,
		FilterDescription:     filterDescription,
		TotalLocations:        totalLocations,
		TotalCountries:        stats.TotalCountries,
		TotalConcerts:         stats.TotalConcerts,
	}

	app.render(w, r, "locations.tmpl", data)
}

// LocationDetail handles individual location pages.
func (app *App) LocationDetail(w http.ResponseWriter, r *http.Request) {
	slug := strings.TrimPrefix(r.URL.Path, "/locations/")
	if slug == "" {
		app.NotFoundError(w, r, "")
		return
	}

	location, found := app.store.LocationBySlug(slug)
	if !found {
		app.NotFoundError(w, r, "Location not found")
		return
	}

	suggestions := app.store.GenerateAllSearchSuggestions()

	data := struct {
		Title        string
		ExtraCSS     string
		ExtraJS      string
		Suggestions  []data.SearchSuggestion
		Location     data.Location
		Artists      []data.ArtistAtLocation
		PrevLocation *data.Location `json:"prevLocation,omitempty"`
		NextLocation *data.Location `json:"nextLocation,omitempty"`
	}{
		Title:        fmt.Sprintf("%s - Location", location.Name),
		ExtraCSS:     "location_detail.css",
		ExtraJS:      "",
		Suggestions:  suggestions,
		Location:     location,
		Artists:      location.Artists,
		PrevLocation: nil, // Could be implemented later for location navigation
		NextLocation: nil, // Could be implemented later for location navigation
	}

	app.render(w, r, "location_detail.tmpl", data)
}

// ============================================================================
// SEARCH
// ============================================================================

// Search handles search requests for artists.
func (app *App) Search(w http.ResponseWriter, r *http.Request) {
	// Validate path using centralized utility
	if !app.validateExactPath(w, r, "/search") {
		return
	}

	var searchQuery string
	var appliedFilters data.ArtistFilterParams
	var searchResults data.SearchResult

	// Handle search submission
	if r.Method == http.MethodPost {
		if !app.parseFormOrError(w, r) {
			return
		}

		searchQuery = strings.TrimSpace(r.FormValue("q"))
		// Extract search term from datalist format "Name - type" if applicable
		searchQuery = extractSearchTerm(searchQuery)
		appliedFilters = parseArtistFilterParams(r)

		searchParams := data.SearchParams{
			Query:   searchQuery,
			Filters: appliedFilters,
		}
		searchResults = app.store.SearchArtists(searchParams)
	}

	filterOptions := app.store.GetArtistFilterOptions()

	// Generate all search suggestions for datalist
	allSuggestions := app.store.GenerateAllSearchSuggestions()

	data := struct {
		Title          string
		ExtraCSS       string
		ExtraJS        string
		Suggestions    []data.SearchSuggestion
		Query          string
		Results        data.SearchResult
		FilterOptions  data.ArtistFilterOptions
		AppliedFilters data.ArtistFilterParams
		IsSearch       bool
	}{
		Title:          "Search",
		ExtraCSS:       "search.css",
		ExtraJS:        "",
		Suggestions:    allSuggestions, // Use cached suggestions
		Query:          searchQuery,
		Results:        searchResults,
		FilterOptions:  filterOptions,
		AppliedFilters: appliedFilters,
		IsSearch:       r.Method == http.MethodPost && searchQuery != "",
	}

	app.render(w, r, "search.tmpl", data)
}

// SuggestionsAPI provides search suggestions for autocomplete functionality.
func (app *App) SuggestionsAPI(w http.ResponseWriter, r *http.Request) {
	query := strings.TrimSpace(r.URL.Query().Get("q"))

	// Use optimized filtering with reasonable limits
	const maxSuggestions = 15 // Limit to avoid overwhelming the UI
	matchingSuggestions := app.store.FilterSearchSuggestions(query, maxSuggestions)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(matchingSuggestions)
}

// ============================================================================
// HEALTH CHECK
// ============================================================================

// Health provides a health check endpoint.
func (app *App) Health(w http.ResponseWriter, r *http.Request) {
	response := map[string]any{
		"status":    "healthy",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"stats":     app.store.Stats(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// ============================================================================
// DEVELOPER TOOLS
// ============================================================================

// DevIndex renders a small developer page with quick links.
func (app *App) DevIndex(w http.ResponseWriter, r *http.Request) {
	links := []struct{ Href, Text string }{
		{"/dev/panic", "Trigger Panic (/dev/panic)"},
		{"/dev/404", "Simulate 404 (/dev/404)"},
		{"/dev/500", "Simulate 500 (/dev/500)"},
		{"/dev/tmpl-error", "Simulate Template Error (/dev/tmpl-error)"},
		{"/health", "Health Check (/health)"},
	}

	suggestions := app.store.GenerateAllSearchSuggestions()

	data := struct {
		Title       string
		ExtraCSS    string
		ExtraJS     string
		Suggestions []data.SearchSuggestion
		Links       []struct{ Href, Text string }
	}{
		Title:       "Developer Tools",
		ExtraCSS:    "dev.css",
		ExtraJS:     "",
		Suggestions: suggestions,
		Links:       links,
	}

	app.render(w, r, "dev.tmpl", data)
}

// DevPanic is a development endpoint to test panic recovery.
func (app *App) DevPanic(w http.ResponseWriter, r *http.Request) {
	panic("Development panic triggered")
}

// Dev404 is a development endpoint to test 404 error template.
func (app *App) Dev404(w http.ResponseWriter, r *http.Request) {
	// Simulate a realistic 404 by mutating a shallow copy of the request
	// so that template rendering sees a non-existent requested URL.
	// We keep the original request untouched and pass the modified copy
	// to the Home handler which will validate the path and trigger a 404.
	nr := new(http.Request)
	*nr = *r
	// Ensure method is GET and set a path that we know doesn't exist in the router
	nr.Method = http.MethodGet
	nr.URL.Path = "/this-page-does-not-exist"

	// Call Home with the modified request so the Error template is rendered
	// using the realistic requested URL stored in nr.URL.Path.
	app.Home(w, nr)
}

// Dev500 is a development endpoint to test 500 error template.
func (app *App) Dev500(w http.ResponseWriter, r *http.Request) {
	app.Error(w, r, http.StatusInternalServerError, "This is a simulated 500 error.")
}

// Dev500Tmpl is a development endpoint to test template failure.
func (app *App) Dev500Tmpl(w http.ResponseWriter, r *http.Request) {
	// To simulate a template error, we can try to render a template that doesn't exist.
	app.render(w, r, "nonexistent.tmpl", nil)
}

// ============================================================================
// ERROR HANDLING
// ============================================================================

// Error handles all errors (4xx and 5xx) in a centralized way.
func (app *App) Error(w http.ResponseWriter, r *http.Request, status int, message string) {
	data := struct {
		Title        string
		ExtraCSS     string
		ExtraJS      string
		Suggestions  []data.SearchSuggestion
		ErrorCode    int
		RequestedURL string
		Message      string
		Timestamp    string
	}{
		Title:        fmt.Sprintf("%d %s", status, http.StatusText(status)),
		ExtraCSS:     "errors.css",
		ExtraJS:      "",
		Suggestions:  nil, // Error pages don't need search suggestions
		ErrorCode:    status,
		RequestedURL: r.URL.Path,
		Message:      message,
		Timestamp:    time.Now().Format("2006-01-02 15:04:05"),
	}

	app.render(w, r, "error.tmpl", data, status)
}

// NotFoundError sends a 404 error response.
func (app *App) NotFoundError(w http.ResponseWriter, r *http.Request, message string) {
	if message == "" {
		message = "Page not found"
	}
	app.Error(w, r, http.StatusNotFound, message)
}

// BadRequestError sends a standardized 400 error response.
func (app *App) BadRequestError(w http.ResponseWriter, r *http.Request, message string) {
	if message == "" {
		message = "Bad request"
	}
	app.Error(w, r, http.StatusBadRequest, message)
}

// validateExactPath checks if request path matches expected path.
func (app *App) validateExactPath(w http.ResponseWriter, r *http.Request, expectedPath string) bool {
	if r.URL.Path != expectedPath {
		app.NotFoundError(w, r, "")
		return false
	}
	return true
}

// parseFormOrError parses form data and handles errors.
func (app *App) parseFormOrError(w http.ResponseWriter, r *http.Request) bool {
	if err := r.ParseForm(); err != nil {
		app.BadRequestError(w, r, "Failed to parse form data")
		return false
	}
	return true
}
