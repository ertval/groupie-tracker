package web

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"groupie-tracker/internal/data"
	"groupie-tracker/internal/view"
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
	featuredArtists := getRandomArtists(artists, 8)

	page := view.NewHomePage(app.store, featuredArtists)
	app.render(w, r, "home.tmpl", page)
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
	var appliedFilters data.ArtistFilterParams
	totalArtists := len(artists)
	isFiltered := false

	// If POST request, parse form data and apply filters
	if r.Method == http.MethodPost {
		if !app.parseFormOrError(w, r) {
			return
		}

		appliedFilters = parseArtistFilterParams(r)
		artists = app.store.FilterArtists(appliedFilters)
		isFiltered = true
	}

	// Sort artists by concert count (descending) for main display
	sort.Slice(artists, func(i, j int) bool {
		return artists[i].ConcertCount() > artists[j].ConcertCount()
	})

	page := view.NewArtistListPage(app.store, artists, filterOptions, appliedFilters, isFiltered, totalArtists)
	app.render(w, r, "artists.tmpl", page)
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

	page := view.NewArtistDetailPage(app.store, artist, prevArtist, nextArtist)
	app.render(w, r, "artist_detail.tmpl", page)
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
	var appliedFilters data.LocationFilterParams
	totalLocations := len(locations)

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

	page := view.NewLocationListPage(app.store, locations, filterOptions, appliedFilters, isFiltered, filterDescription, totalLocations)
	app.render(w, r, "locations.tmpl", page)
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

	page := view.NewLocationDetailPage(app.store, location, location.Artists)
	app.render(w, r, "location_detail.tmpl", page)
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
	isSearch := false

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
		isSearch = searchQuery != ""
	}

	filterOptions := app.store.GetArtistFilterOptions()
	page := view.NewSearchPage(app.store, searchQuery, searchResults, filterOptions, appliedFilters, isSearch)
	app.render(w, r, "search.tmpl", page)
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
	response := view.NewHealthResponse(app.store)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// ============================================================================
// DEVELOPER TOOLS
// ============================================================================

// DevIndex renders a small developer page with quick links.
func (app *App) DevIndex(w http.ResponseWriter, r *http.Request) {
	page := view.NewDevPage(app.store)
	app.render(w, r, "dev.tmpl", page)
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
	page := view.NewErrorPage(status, r.URL.Path, message)
	app.render(w, r, "error.tmpl", page, status)
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
