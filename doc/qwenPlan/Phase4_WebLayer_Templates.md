# Phase 4: Web Layer & Template Simplification - Implementation Guide

## Overview
This phase focuses on simplifying the web layer and template system by creating reusable view models, slimming down handlers, and simplifying template utilities. The goal is to create cleaner, more maintainable web layer code with reduced duplication.

## Step-by-Step Implementation

### Step 1: Create Shared View Models
**File to create:** `internal/web/views.go`

```go
package web

import "groupie-tracker/internal/data"

// BasePage represents common fields needed across all pages
type BasePage struct {
	Title       string                    `json:"title"`
	Description string                    `json:"description,omitempty"`
	ExtraCSS    string                    `json:"extraCSS,omitempty"`
	ExtraJS     string                    `json:"extraJS,omitempty"`
	Suggestions []data.SimpleSuggestion   `json:"suggestions"`
}

// HomePage represents data for the home page
type HomePage struct {
	BasePage
	Artists        []*data.Artist  `json:"artists"`
	TotalMembers   int             `json:"totalMembers"`
	TotalLocations int             `json:"totalLocations"`
}

// ArtistsPage represents data for the artists list page
type ArtistsPage struct {
	BasePage
	Artists        []*data.Artist              `json:"artists"`
	FilterOptions  data.ArtistFilterOptions    `json:"filterOptions"`
	AppliedFilters data.ArtistFilterParams     `json:"appliedFilters"`
	IsFiltered     bool                        `json:"isFiltered"`
	TotalArtists   int                         `json:"totalArtists"`
}

// ArtistDetailPage represents data for the artist detail page
type ArtistDetailPage struct {
	BasePage
	Artist     *data.Artist  `json:"artist"`
	PrevArtist *data.Artist  `json:"prevArtist,omitempty"`
	NextArtist *data.Artist  `json:"nextArtist,omitempty"`
}

// LocationsPage represents data for the locations list page
type LocationsPage struct {
	BasePage
	Locations             []data.Location                `json:"locations"`
	LocationFilterOptions data.LocationFilterOptions     `json:"locationFilterOptions"`
	AppliedFilters        data.LocationFilterParams      `json:"appliedFilters"`
	IsFiltered            bool                           `json:"isFiltered"`
	FilterDescription     string                         `json:"filterDescription"`
	TotalLocations        int                            `json:"totalLocations"`
	TotalCountries        int                            `json:"totalCountries"`
	TotalConcerts         int                            `json:"totalConcerts"`
}

// LocationDetailPage represents data for the location detail page
type LocationDetailPage struct {
	BasePage
	Location data.Location              `json:"location"`
	Artists  []data.ArtistAtLocation    `json:"artists"`
}

// SearchPage represents data for the search results page
type SearchPage struct {
	BasePage
	Query          string                 `json:"query"`
	Results        data.SearchResult      `json:"results"`
	FilterOptions  data.ArtistFilterOptions `json:"filterOptions"`
	AppliedFilters data.ArtistFilterParams  `json:"appliedFilters"`
	IsSearch       bool                   `json:"isSearch"`
}

// ErrorPage represents data for error pages
type ErrorPage struct {
	BasePage
	ErrorCode    int    `json:"errorCode"`
	RequestedURL string `json:"requestedURL"`
	Message      string `json:"message"`
	Timestamp    string `json:"timestamp"`
}

// DevPage represents data for developer tools page
type DevPage struct {
	BasePage
	Links []struct {
		Href, Text string
	} `json:"links"`
}
```

### Step 2: Add View Helper Functions
**File to continue:** `internal/web/views.go`

```go
// NewBasePage creates a new base page with common elements
func (app *App) NewBasePage(title string, extraCSS, extraJS string) BasePage {
	return BasePage{
		Title:       title,
		ExtraCSS:    extraCSS,
		ExtraJS:     extraJS,
		Suggestions: app.store.GenerateAllSearchSuggestions(),
	}
}

// NewHomePage creates a homepage with all required data
func (app *App) NewHomePage() HomePage {
	artists := app.store.Artists()
	stats := app.store.Stats()
	suggestions := app.store.GenerateAllSearchSuggestions()
	
	// Get 8 random artists for homepage display
	homeArtists := getRandomArtists(artists, 8)
	
	return HomePage{
		BasePage: BasePage{
			Title:       "Home",
			ExtraCSS:    "home.css",
			ExtraJS:     "",
			Suggestions: suggestions,
		},
		Artists:        homeArtists,
		TotalMembers:   stats.TotalMembers,
		TotalLocations: stats.TotalLocations,
	}
}

// NewArtistsPage creates an artists page with optional filtering
func (app *App) NewArtistsPage(filters data.ArtistFilterParams) ArtistsPage {
	var artists []*data.Artist
	var isFiltered bool
	
	if isEmptyFilter(filters) {
		artists = app.store.Artists()
		isFiltered = false
	} else {
		artists = app.store.FilterArtists(filters)
		isFiltered = true
	}
	
	filterOptions := app.store.GetArtistFilterOptions()
	suggestions := app.store.GenerateAllSearchSuggestions()
	
	return ArtistsPage{
		BasePage: BasePage{
			Title:       "Artists",
			ExtraCSS:    "artists.css",
			ExtraJS:     "",
			Suggestions: suggestions,
		},
		Artists:        artists,
		FilterOptions:  filterOptions,
		AppliedFilters: filters,
		IsFiltered:     isFiltered,
		TotalArtists:   len(app.store.Artists()),
	}
}

// NewArtistDetailPage creates an artist detail page
func (app *App) NewArtistDetailPage(artist *data.Artist) ArtistDetailPage {
	// Get navigation artists using on-demand lookup
	prevArtist, nextArtist := app.store.GetAdjacentArtists(artist.ID)
	suggestions := app.store.GenerateAllSearchSuggestions()

	return ArtistDetailPage{
		BasePage: BasePage{
			Title:       artist.Name,
			ExtraCSS:    "artist_detail.css",
			ExtraJS:     "",
			Suggestions: suggestions,
		},
		Artist:     artist,
		PrevArtist: prevArtist,
		NextArtist: nextArtist,
	}
}

// NewLocationsPage creates a locations page with optional filtering
func (app *App) NewLocationsPage(filters data.LocationFilterParams) LocationsPage {
	var locations []data.Location
	var isFiltered bool
	
	if isEmptyLocationFilter(filters) {
		locations = app.store.Locations()
		isFiltered = false
	} else {
		locations = app.store.FilterLocations(filters)
		isFiltered = true
	}
	
	stats := app.store.Stats()
	filterOptions := app.store.GetLocationFilterOptions()
	suggestions := app.store.GenerateAllSearchSuggestions()
	
	// Generate filter description
	filterDescription := ""
	if isFiltered {
		if len(filters.Countries) > 0 {
			if len(filters.Countries) == 1 {
				filterDescription = filters.Countries[0]
			} else {
				filterDescription = "Multiple Countries"
			}
		} else {
			filterDescription = "Filters Applied"
		}
	}
	
	return LocationsPage{
		BasePage: BasePage{
			Title:       "Locations",
			ExtraCSS:    "locations.css",
			ExtraJS:     "",
			Suggestions: suggestions,
		},
		Locations:             locations,
		LocationFilterOptions: filterOptions,
		AppliedFilters:        filters,
		IsFiltered:            isFiltered,
		FilterDescription:     filterDescription,
		TotalLocations:        len(app.store.Locations()),
		TotalCountries:        stats.TotalCountries,
		TotalConcerts:         stats.TotalConcerts,
	}
}

// NewLocationDetailPage creates a location detail page
func (app *App) NewLocationDetailPage(location data.Location) LocationDetailPage {
	suggestions := app.store.GenerateAllSearchSuggestions()

	return LocationDetailPage{
		BasePage: BasePage{
			Title:       fmt.Sprintf("%s - Location", location.Name),
			ExtraCSS:    "location_detail.css",
			ExtraJS:     "",
			Suggestions: suggestions,
		},
		Location: location,
		Artists:  location.ArtistsAtLocation(),
	}
}

// NewSearchPage creates a search results page
func (app *App) NewSearchPage(query string, filters data.ArtistFilterParams) SearchPage {
	var searchResults data.SearchResult
	var isSearch bool

	if query != "" {
		searchResults = app.store.SearchArtists(query, filters)
		isSearch = true
	} else if !isEmptyFilter(filters) {
		// Only filters applied, no query
		artists := app.store.FilterArtists(filters)
		searchResults = data.SearchResult{
			Artists:      artists,
			Query:        query,
			TotalResults: len(artists),
		}
		isSearch = false
	} else {
		// No query, no filters - return empty results
		searchResults = data.SearchResult{
			Artists:      []*data.Artist{},
			Query:        query,
			TotalResults: 0,
		}
		isSearch = false
	}

	filterOptions := app.store.GetArtistFilterOptions()
	suggestions := app.store.GenerateAllSearchSuggestions()

	return SearchPage{
		BasePage: BasePage{
			Title:       "Search",
			ExtraCSS:    "search.css",
			ExtraJS:     "",
			Suggestions: suggestions, // Use cached suggestions
		},
		Query:          query,
		Results:        searchResults,
		FilterOptions:  filterOptions,
		AppliedFilters: filters,
		IsSearch:       isSearch,
	}
}

// isEmptyFilter checks if filter parameters are empty.
func isEmptyFilter(filters data.ArtistFilterParams) bool {
	return (filters.CreationYear.Min == nil && filters.CreationYear.Max == nil) &&
		(filters.FirstAlbumYear.Min == nil && filters.FirstAlbumYear.Max == nil) &&
		len(filters.MemberCounts) == 0 &&
		len(filters.Countries) == 0
}

// isEmptyLocationFilter checks if location filter parameters are empty.
func isEmptyLocationFilter(filters data.LocationFilterParams) bool {
	return (filters.ConcertCount.Min == nil && filters.ConcertCount.Max == nil) &&
		(filters.ArtistCount.Min == nil && filters.ArtistCount.Max == nil) &&
		(filters.YearRange.Min == nil && filters.YearRange.Max == nil) &&
		len(filters.Countries) == 0
}
```

### Step 3: Simplify HTTP Handlers
**File to modify:** `internal/web/handlers.go`

```go
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

	homePage := app.NewHomePage()
	app.render(w, r, "home.tmpl", homePage)
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

	var appliedFilters data.ArtistFilterParams

	// If POST request, parse form data and apply filters
	if r.Method == http.MethodPost {
		if !app.parseFormOrError(w, r) {
			return
		}
		appliedFilters = parseArtistFilterParams(r)
	}

	artistsPage := app.NewArtistsPage(appliedFilters)
	
	// Sort artists by concert count (descending) for main display
	sort.Slice(artistsPage.Artists, func(i, j int) bool {
		return artistsPage.Artists[i].ConcertCount() > artistsPage.Artists[j].ConcertCount()
	})

	app.render(w, r, "artists.tmpl", artistsPage)
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

	artistDetailPage := app.NewArtistDetailPage(artist)
	app.render(w, r, "artist_detail.tmpl", artistDetailPage)
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

	var appliedFilters data.LocationFilterParams

	// If POST request, parse form data and apply filters
	if r.Method == http.MethodPost {
		if !app.parseFormOrError(w, r) {
			return
		}
		appliedFilters = parseLocationFilterParams(r)
	}

	locationsPage := app.NewLocationsPage(appliedFilters)
	app.render(w, r, "locations.tmpl", locationsPage)
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

	locationDetailPage := app.NewLocationDetailPage(location)
	app.render(w, r, "location_detail.tmpl", locationDetailPage)
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

	// Handle search submission
	if r.Method == http.MethodPost {
		if !app.parseFormOrError(w, r) {
			return
		}

		searchQuery = strings.TrimSpace(r.FormValue("q"))
		// Extract search term from datalist format "Name - type" if applicable
		searchQuery = extractSearchTerm(searchQuery)
		appliedFilters = parseArtistFilterParams(r)
	}

	searchPage := app.NewSearchPage(searchQuery, appliedFilters)
	app.render(w, r, "search.tmpl", searchPage)
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

	devPage := DevPage{
		BasePage: BasePage{
			Title:       "Developer Tools",
			ExtraCSS:    "dev.css",
			ExtraJS:     "",
			Suggestions: app.store.GenerateAllSearchSuggestions(),
		},
		Links: links,
	}

	app.render(w, r, "dev.tmpl", devPage)
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
	errorPage := ErrorPage{
		BasePage: BasePage{
			Title:    fmt.Sprintf("%d %s", status, http.StatusText(status)),
			ExtraCSS: "errors.css",
			ExtraJS:  "",
		},
		ErrorCode:    status,
		RequestedURL: r.URL.Path,
		Message:      message,
		Timestamp:    time.Now().Format("2006-01-02 15:04:05"),
	}

	app.render(w, r, "error.tmpl", errorPage, status)
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
```

### Step 4: Simplify Template Loading and Rendering
**File to modify:** `internal/web/templates.go`

```go
// render executes a template and sends the response.
// Now works with the new view models.
func (app *App) render(w http.ResponseWriter, r *http.Request, name string, data interface{}, status ...int) {
	code := http.StatusOK
	if len(status) > 0 {
		code = status[0]
	}

	tmpl, ok := app.templates[name]
	if !ok {
		// Prevent infinite recursion if error template itself is missing
		if name == "error.tmpl" {
			log.Printf("FATAL: error.tmpl is missing")
			http.Error(w, "500 Internal Server Error - Error template not found", http.StatusInternalServerError)
			return
		}
		app.Error(w, r, http.StatusInternalServerError, fmt.Sprintf("Template %s not found", name))
		return
	}

	// Use buffer to catch template execution errors before sending response
	buf := new(bytes.Buffer)
	if err := tmpl.ExecuteTemplate(buf, "base", data); err != nil {
		// Handle error template execution failure gracefully
		if name == "error.tmpl" {
			log.Printf("Error executing error template: %v", err)
			http.Error(w, "500 Internal Server Error - Failed to execute error template", http.StatusInternalServerError)
			return
		}
		app.Error(w, r, http.StatusInternalServerError, err.Error())
		return
	}

	// Only send response after successful template execution
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(code)
	buf.WriteTo(w)
}

// loadTemplates compiles and caches all HTML templates.
// Simplified with clearer organization.
func (app *App) loadTemplates() {
	app.templates = make(map[string]*template.Template)

	// Custom template functions for common operations
	funcMap := template.FuncMap{
		"add":   func(a, b int) int { return a + b },
		"sub":   func(a, b int) int { return a - b },
		"join":  func(items []string, sep string) string { return strings.Join(items, sep) },
		"upper": func(s string) string { return strings.ToUpper(s) },
		"title": func(s string) string {
			words := strings.Fields(strings.ReplaceAll(s, "-", " "))
			for i, word := range words {
				if len(word) > 0 {
					words[i] = strings.ToUpper(string(word[0])) + strings.ToLower(word[1:])
				}
			}
			return strings.Join(words, " ")
		},
		"contains": func(slice interface{}, item interface{}) bool {
			switch s := slice.(type) {
			case []int:
				if i, ok := item.(int); ok {
					for _, v := range s {
						if v == i {
							return true
						}
					}
				}
			case []string:
				if str, ok := item.(string); ok {
					for _, v := range s {
						if v == str {
							return true
						}
					}
				}
			}
			return false
		},
		// Add helper to get concert count from artist
		"concertCount": func(artist *data.Artist) int {
			if artist == nil {
				return 0
			}
			return artist.ConcertCount()
		},
		// Add helper to get member count from artist
		"memberCount": func(artist *data.Artist) int {
			if artist == nil {
				return 0
			}
			return artist.MemberCount()
		},
		// Add helper to get artist's slug
		"artistSlug": func(artist *data.Artist) string {
			if artist == nil {
				return ""
			}
			return artist.Slug()
		},
		// Add helper to get location's slug
		"locationSlug": func(location data.Location) string {
			return location.Slug()
		},
	}

	const templateDir = "templates"
	baseTmplPath := filepath.Join(templateDir, "base.tmpl")

	if _, err := os.Stat(baseTmplPath); err != nil {
		log.Fatalf("Failed to find base template at %s: %v", baseTmplPath, err)
	}

	// Discover all template files
	pages, err := filepath.Glob(filepath.Join(templateDir, "*.tmpl"))
	if err != nil {
		log.Fatalf("Failed to glob templates: %v", err)
	}

	// Compile each template with base template for inheritance
	for _, page := range pages {
		name := filepath.Base(page)
		if name == "base.tmpl" {
			continue // Skip base template as it's included in each page
		}

		ts, err := template.New(name).Funcs(funcMap).ParseFiles(baseTmplPath, page)
		if err != nil {
			log.Fatalf("Failed to parse template %s: %v", name, err)
		}

		app.templates[name] = ts
	}
	
	log.Printf("Compiled %d templates successfully", len(app.templates))
}
```

### Step 5: Add Request Processing Utilities
**File to continue:** `internal/web/templates.go` (or create `internal/web/utils.go`)

```go
// Utility functions for common request processing

// requireMethod validates that the incoming request uses one of the allowed HTTP methods.
func requireMethod(w http.ResponseWriter, r *http.Request, allowedMethods ...string) bool {
	allowed := slices.Contains(allowedMethods, r.Method)

	if !allowed {
		// Build Allow header with comma-separated list of valid methods (required by HTTP spec)
		allowHeader := strings.Join(allowedMethods, ", ")
		w.Header().Set("Allow", allowHeader)
		return false
	}
	return true
}

// parseFormWithError handles form parsing with error response
func parseFormWithError(w http.ResponseWriter, r *http.Request, errorFunc func(string)) bool {
	if err := r.ParseForm(); err != nil {
		errorFunc("Failed to parse form data")
		return false
	}
	return true
}

// parseIntPtr parses integer form field and returns pointer.
func parseIntPtr(r *http.Request, fieldName string) *int {
	if str := r.FormValue(fieldName); str != "" {
		if val, err := strconv.Atoi(str); err == nil {
			return &val
		}
	}
	return nil
}

// parseIntSlice parses multiple checkbox values into integer slice.
func parseIntSlice(r *http.Request, fieldName string) []int {
	var results []int
	if values := r.Form[fieldName]; len(values) > 0 {
		for _, valueStr := range values {
			if value, err := strconv.Atoi(valueStr); err == nil {
				results = append(results, value)
			}
		}
	}
	return results
}

// parseStringSlice parses multiple form values into string slice.
func parseStringSlice(r *http.Request, fieldName string) []string {
	if values := r.Form[fieldName]; len(values) > 0 {
		return values
	}
	return nil
}
```

### Step 6: Simplify Middleware
**File to modify:** `internal/web/middleware.go`

```go
package web

import (
	"log"
	"net/http"
	"time"
)

// withMiddleware assembles the complete middleware chain for all HTTP requests.
// Chain order (innermost to outermost): secureHeaders → recovery → logging
// This order ensures security headers are set first, panics are caught, and all requests are logged.
func withMiddleware(next http.Handler) http.Handler {
	return withLogging(withRecovery(withSecureHeaders(next)))
}

// withRecovery catches panics in handlers and converts them to 500 errors instead of crashing the server.
// Logs panic details for debugging while returning a generic error to the client for security.
func withRecovery(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil { // Catch any panic from downstream handlers
				log.Printf("Panic recovered: %v", err) // Log for debugging (includes stack trace in production logs)
				w.Header().Set("Content-Type", "text/plain; charset=utf-8")
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte("500 Internal Server Error")) // Generic message to avoid leaking internal details
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// withLogging logs each HTTP request with method, path, and response time for monitoring and debugging.
// Time measurement starts before handler execution and completes after response is sent.
func withLogging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()                                             // Record request start time
		next.ServeHTTP(w, r)                                            // Execute the actual handler
		log.Printf("%s %s %v", r.Method, r.URL.Path, time.Since(start)) // Log after response: "GET /artists 15.2ms"
	})
}

// withSecureHeaders injects standard security headers into every HTTP response to mitigate common web vulnerabilities.
// Headers protect against: content sniffing, clickjacking, XSS, and referrer leakage.
// CSP is intentionally omitted to allow flexibility with external resources (images, fonts, CDNs).
func withSecureHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Referrer-Policy", "origin-when-cross-origin") // Send only origin in cross-origin referrers (privacy)
		w.Header().Set("X-Content-Type-Options", "nosniff")           // Prevent MIME type sniffing (security)
		w.Header().Set("X-Frame-Options", "deny")                     // Prevent clickjacking by blocking iframe embedding
		w.Header().Set("X-XSS-Protection", "0")                       // Disable legacy XSS filter (modern CSP is better, but not set here)
		// Content-Security-Policy intentionally not set - allows external images/fonts/APIs without restriction
		next.ServeHTTP(w, r)
	})
}

// restrictMethod validates that the incoming request uses one of the allowed HTTP methods.
// Returns 405 Method Not Allowed with proper Allow header if method is not permitted.
// This is a method on App to allow access to App.Error for consistent error responses.
func (app *App) restrictMethod(handler http.HandlerFunc, allowedMethods ...string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Use the utility function
		if !requireMethod(w, r, allowedMethods...) {
			app.Error(w, r, http.StatusMethodNotAllowed, "Method not allowed")
			return
		}
		handler(w, r) // Method is allowed, proceed with handler execution
	}
}
```

### Step 7: Update Routes to Use Simplified Structure
**File to modify:** `internal/web/routes.go`

```go
package web

import (
	"net/http"
)

// createServeMux initializes and configures the HTTP router with all application routes.
// Returns a configured *http.ServeMux with handlers for static files, API endpoints,
// web pages, and development tools. Routes are organized by functionality for clarity.
func (app *App) createServeMux() *http.ServeMux {
	router := http.NewServeMux()

	// Static assets: CSS, JS, images, and favicon
	router.HandleFunc("/static/", app.restrictMethod(app.StaticFiles, "GET", "HEAD"))
	router.HandleFunc("/favicon.ico", app.restrictMethod(app.StaticFiles, "GET", "HEAD"))

	// Health check endpoint for monitoring
	router.HandleFunc("/health", app.restrictMethod(app.Health, "GET"))

	// API endpoints
	router.HandleFunc("/api/suggestions", app.restrictMethod(app.SuggestionsAPI, "GET"))

	// Search endpoints (supports both GET and POST)
	router.HandleFunc("/search", app.restrictMethod(app.Search, "GET", "POST"))

	// Development tools (only active in dev mode)
	router.HandleFunc("/dev", app.restrictMethod(app.DevIndex, "GET"))
	router.HandleFunc("/dev/panic", app.DevPanic) // No method guard - allows any method for testing
	router.HandleFunc("/dev/404", app.Dev404)
	router.HandleFunc("/dev/500", app.Dev500)
	router.HandleFunc("/dev/tmpl-error", app.Dev500Tmpl)

	// Main application pages with filter support
	router.HandleFunc("/artists", app.restrictMethod(app.Artists, "GET", "POST"))
	router.HandleFunc("/artists/", app.restrictMethod(app.ArtistDetail, "GET"))
	router.HandleFunc("/locations", app.restrictMethod(app.Locations, "GET", "POST"))
	router.HandleFunc("/locations/", app.restrictMethod(app.LocationDetail, "GET"))

	// Home page (catch-all root handler)
	router.HandleFunc("/", app.restrictMethod(app.Home, "GET"))

	return router
}
```

### Step 8: Update Helper Functions in templates.go
**File to continue:** `internal/web/templates.go`

```go
// extractSearchTerm extracts search term from datalist suggestion format.
func extractSearchTerm(input string) string {
	if input == "" {
		return input
	}

	// Check if input matches datalist format "term - type"
	if lastDash := strings.LastIndex(input, " - "); lastDash != -1 {
		term := strings.TrimSpace(input[:lastDash])
		if term != "" {
			return term
		}
	}

	return input
}

// getRandomArtists shuffles the provided artists slice and returns up to maxCount random artists.
// This utility function encapsulates the randomization logic to keep handlers clean.
func getRandomArtists(artists []*data.Artist, maxCount int) []*data.Artist {
	if len(artists) == 0 {
		return artists
	}

	// Create a copy to avoid modifying the original slice
	shuffled := make([]*data.Artist, len(artists))
	copy(shuffled, artists)

	// Shuffle the copy
	rand.Seed(time.Now().UnixNano()) // Ensure proper seeding
	rand.Shuffle(len(shuffled), func(i, j int) {
		shuffled[i], shuffled[j] = shuffled[j], shuffled[i]
	})

	// Limit to maxCount
	if len(shuffled) > maxCount {
		shuffled = shuffled[:maxCount]
	}

	return shuffled
}
```

## Testing Strategy for Phase 4
1. Update existing web layer tests to work with new view models
2. Verify all handlers still return the expected data to the templates
3. Test that all URLs still work as expected
4. Ensure error handling still works properly
5. Verify that template rendering still works with the new data structures
6. Test all navigation between pages still works

## Rollout Considerations
- The simplified web layer should be more maintainable and easier to understand
- All existing functionality should be preserved
- Templates will need to be updated to work with the new view model structures
- The API between handlers and templates has changed, so thorough testing is important
- Consider backward compatibility if external systems depend on specific response formats