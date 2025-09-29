package server

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

// Home handles the home page.
func (s *Server) Home(w http.ResponseWriter, r *http.Request) {
	// Validate path
	if r.URL.Path != "/" {
		s.Error(w, r, http.StatusNotFound, "Page not found")
		return
	}

	artists := s.artists.GetArtists()
	stats := s.stats.GetStats()

	// Get 8 random artists for homepage display
	artists = getRandomArtists(artists, 8)

	data := struct {
		BaseTemplateData
		Artists        []data.Artist
		TotalMembers   int
		TotalLocations int
	}{
		BaseTemplateData: s.NewBaseTemplateData("Home", "home.css"),
		Artists:          artists,
		TotalMembers:     stats["total_members"],
		TotalLocations:   stats["total_locations"],
	}

	s.render(w, r, "home.tmpl", data)
}

// Artists handles the artists listing page.
func (s *Server) Artists(w http.ResponseWriter, r *http.Request) {
	// Validate path for both GET and POST
	if r.URL.Path != "/artists" {
		s.Error(w, r, http.StatusNotFound, "Page not found")
		return
	}

	artists := s.artists.GetArtists()
	filterOptions := s.artists.GetArtistFilterOptions()
	var appliedFilters data.ArtistFilterParams
	totalArtists := len(artists)

	// If POST request, parse form data and apply filters
	if r.Method == http.MethodPost {
		if err := r.ParseForm(); err != nil {
			s.Error(w, r, http.StatusBadRequest, "Failed to parse form data")
			return
		}

		appliedFilters = parseArtistFilterParams(r)
		artists = s.artists.FilterArtists(appliedFilters)
	}

	// Sort artists by concert count (descending) for main display
	sort.Slice(artists, func(i, j int) bool {
		return artists[i].ConcertCount > artists[j].ConcertCount
	})

	data := struct {
		BaseTemplateData
		Artists        []data.Artist
		FilterOptions  data.ArtistFilterOptions
		AppliedFilters data.ArtistFilterParams
		IsFiltered     bool
		TotalArtists   int
	}{
		BaseTemplateData: s.NewBaseTemplateData("Artists", "artists.css"),
		Artists:          artists,
		FilterOptions:    filterOptions,
		AppliedFilters:   appliedFilters,
		IsFiltered:       r.Method == http.MethodPost,
		TotalArtists:     totalArtists,
	}

	s.render(w, r, "artists.tmpl", data)
}

// ArtistDetail handles individual artist pages.
func (s *Server) ArtistDetail(w http.ResponseWriter, r *http.Request) {
	// Validate path
	path := strings.TrimPrefix(r.URL.Path, "/artists/")
	if path == "" {
		s.Error(w, r, http.StatusNotFound, "Page not found")
		return
	}

	// Try slug first, then ID
	artist, found := s.artists.GetArtistBySlug(path)
	if !found {
		if id, err := strconv.Atoi(path); err == nil {
			artist, found = s.artists.GetArtistByID(id)
		}
		if !found {
			s.Error(w, r, http.StatusNotFound, "Artist not found")
			return
		}
	}

	// Get navigation artists using on-demand lookup
	prevArtist, nextArtist := s.artists.GetAdjacentArtists(artist.ID)

	data := struct {
		BaseTemplateData
		Artist     data.Artist
		PrevArtist *data.Artist
		NextArtist *data.Artist
	}{
		BaseTemplateData: s.NewBaseTemplateData(artist.Name, "artist_detail.css"),
		Artist:           artist,
		PrevArtist:       prevArtist,
		NextArtist:       nextArtist,
	}

	s.render(w, r, "artist_detail.tmpl", data)
}

// Search handles search requests for artists.
func (s *Server) Search(w http.ResponseWriter, r *http.Request) {
	// Validate path
	if r.URL.Path != "/search" {
		s.Error(w, r, http.StatusNotFound, "Page not found")
		return
	}

	var searchQuery string
	var appliedFilters data.ArtistFilterParams
	var searchResults data.SearchResult

	// Handle search submission
	if r.Method == http.MethodPost {
		if err := r.ParseForm(); err != nil {
			s.Error(w, r, http.StatusBadRequest, "Failed to parse form data")
			return
		}

		searchQuery = strings.TrimSpace(r.FormValue("q"))
		// Extract search term from datalist format "Name - type" if applicable
		searchQuery = extractSearchTerm(searchQuery)
		appliedFilters = parseArtistFilterParams(r)

		// Perform search
		searchParams := data.SearchParams{
			Query:   searchQuery,
			Filters: appliedFilters,
		}
		searchResults = s.search.SearchArtists(searchParams)
	}

	filterOptions := s.artists.GetArtistFilterOptions()

	// Generate all search suggestions for datalist
	allSuggestions := s.search.GenerateAllSearchSuggestions()

	data := struct {
		BaseTemplateData
		Query          string
		Results        data.SearchResult
		FilterOptions  data.ArtistFilterOptions
		AppliedFilters data.ArtistFilterParams
		IsSearch       bool
	}{
		BaseTemplateData: BaseTemplateData{
			Title:       "Search",
			ExtraCSS:    "search.css",
			ExtraJS:     "",
			Suggestions: allSuggestions,
		},
		Query:          searchQuery,
		Results:        searchResults,
		FilterOptions:  filterOptions,
		AppliedFilters: appliedFilters,
		IsSearch:       r.Method == http.MethodPost && searchQuery != "",
	}

	s.render(w, r, "search.tmpl", data)
}

// Locations handles the locations listing page.
func (s *Server) Locations(w http.ResponseWriter, r *http.Request) {
	// Validate path for both GET and POST
	if r.URL.Path != "/locations" {
		s.Error(w, r, http.StatusNotFound, "Page not found")
		return
	}

	locations := s.locations.GetLocations()
	filterOptions := s.repo.GetLocationFilterOptions()
	var appliedFilters data.LocationFilterParams
	totalLocations := len(locations)
	stats := s.stats.GetStats()

	// If POST request, parse form data and apply filters
	if r.Method == http.MethodPost {
		if err := r.ParseForm(); err != nil {
			s.Error(w, r, http.StatusBadRequest, "Failed to parse form data")
			return
		}

		appliedFilters = parseLocationFilterParams(r)
		locations = s.repo.FilterLocations(appliedFilters)
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
		BaseTemplateData
		Locations             []data.Location
		LocationFilterOptions data.LocationFilterOptions
		AppliedFilters        data.LocationFilterParams
		IsFiltered            bool
		FilterDescription     string
		TotalLocations        int
		TotalCountries        int
		TotalConcerts         int
	}{
		BaseTemplateData:      s.NewBaseTemplateData("Locations", "locations.css"),
		Locations:             locations,
		LocationFilterOptions: filterOptions,
		AppliedFilters:        appliedFilters,
		IsFiltered:            isFiltered,
		FilterDescription:     filterDescription,
		TotalLocations:        totalLocations,
		TotalCountries:        stats["total_countries"],
		TotalConcerts:         stats["total_concerts"],
	}

	s.render(w, r, "locations.tmpl", data)
}

// LocationDetail handles individual location pages.
func (s *Server) LocationDetail(w http.ResponseWriter, r *http.Request) {
	slug := strings.TrimPrefix(r.URL.Path, "/locations/")
	if slug == "" {
		s.Error(w, r, http.StatusNotFound, "Page not found")
		return
	}

	location, found := s.locations.GetLocationBySlug(slug)
	if !found {
		s.Error(w, r, http.StatusNotFound, "Location not found")
		return
	}

	data := struct {
		BaseTemplateData
		Location     data.Location
		Artists      []data.ArtistAtLocation
		PrevLocation *data.Location `json:"prevLocation,omitempty"`
		NextLocation *data.Location `json:"nextLocation,omitempty"`
	}{
		BaseTemplateData: s.NewBaseTemplateData(fmt.Sprintf("%s - Location", location.Name), "location_detail.css"),
		Location:         location,
		Artists:          location.Artists,
		PrevLocation:     nil, // Could be implemented later for location navigation
		NextLocation:     nil, // Could be implemented later for location navigation
	}

	s.render(w, r, "location_detail.tmpl", data)
}

// DevIndex renders a small developer page with quick links.
func (s *Server) DevIndex(w http.ResponseWriter, r *http.Request) {
	links := []struct{ Href, Text string }{
		{"/dev/panic", "Trigger Panic (/dev/panic)"},
		{"/dev/404", "Simulate 404 (/dev/404)"},
		{"/dev/500", "Simulate 500 (/dev/500)"},
		{"/dev/tmpl-error", "Simulate Template Error (/dev/tmpl-error)"},
		{"/health", "Health Check (/health)"},
	}

	data := struct {
		BaseTemplateData
		Links []struct{ Href, Text string }
	}{
		BaseTemplateData: s.NewBaseTemplateData("Developer Tools", "dev.css"),
		Links:            links,
	}

	s.render(w, r, "dev.tmpl", data)
}

// Error handles all errors (4xx and 5xx) in a centralized way.
func (s *Server) Error(w http.ResponseWriter, r *http.Request, status int, message string) {
	data := struct {
		BaseTemplateData
		ErrorCode    int
		RequestedURL string
		Message      string
		Timestamp    string
	}{
		BaseTemplateData: BaseTemplateData{
			Title:       fmt.Sprintf("%d %s", status, http.StatusText(status)),
			ExtraCSS:    "errors.css",
			ExtraJS:     "",
			Suggestions: nil, // Error pages don't need search suggestions
		},
		ErrorCode:    status,
		RequestedURL: r.URL.Path,
		Message:      message,
		Timestamp:    time.Now().Format("2006-01-02 15:04:05"),
	}

	s.render(w, r, "error.tmpl", data, status)
}

// Health provides a health check endpoint.
func (s *Server) Health(w http.ResponseWriter, r *http.Request) {
	response := map[string]any{
		"status":    "healthy",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"stats":     s.stats.GetStats(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// SuggestionsAPI provides search suggestions for autocomplete functionality.
func (s *Server) SuggestionsAPI(w http.ResponseWriter, r *http.Request) {
	query := strings.TrimSpace(r.URL.Query().Get("q"))

	// Get all suggestions
	allSuggestions := s.search.GenerateAllSearchSuggestions()

	// If no query, return empty suggestions
	if query == "" {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]data.SearchSuggestion{})
		return
	}

	// Filter suggestions based on query
	var matchingSuggestions []data.SearchSuggestion
	queryLower := strings.ToLower(query)

	for _, suggestion := range allSuggestions {
		if strings.Contains(strings.ToLower(suggestion.Text), queryLower) {
			matchingSuggestions = append(matchingSuggestions, suggestion)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(matchingSuggestions)
}

// DevPanic is a development endpoint to test panic recovery.
func (s *Server) DevPanic(w http.ResponseWriter, r *http.Request) {
	panic("Development panic triggered")
}

// Dev404 is a development endpoint to test 404 error template.
func (s *Server) Dev404(w http.ResponseWriter, r *http.Request) {
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
	s.Home(w, nr)
}

// Dev500 is a development endpoint to test 500 error template.
func (s *Server) Dev500(w http.ResponseWriter, r *http.Request) {
	s.Error(w, r, http.StatusInternalServerError, "This is a simulated 500 error.")
}

// Dev500Tmpl is a development endpoint to test template failure.
func (s *Server) Dev500Tmpl(w http.ResponseWriter, r *http.Request) {
	// To simulate a template error, we can try to render a template that doesn't exist.
	s.render(w, r, "nonexistent.tmpl", nil)
}

func (s *Server) StaticFiles(w http.ResponseWriter, r *http.Request) {
	const staticDir = "static"

	// Handle favicon.ico requests
	if r.URL.Path == "/favicon.ico" {
		target := filepath.Join(staticDir, "favicon.ico")
		if fi, err := os.Stat(target); err != nil || fi.IsDir() {
			s.Error(w, r, http.StatusNotFound, "Favicon not found")
			return
		}
		http.ServeFile(w, r, target)
		return
	}

	// Only allow /static/ prefix
	if !strings.HasPrefix(r.URL.Path, "/static/") {
		s.Error(w, r, http.StatusNotFound, "Not found")
		return
	}

	// Extract relative path and prevent directory traversal
	rel := strings.TrimPrefix(r.URL.Path, "/static/")
	if rel == "" || strings.Contains(rel, "..") || strings.HasPrefix(rel, "/") {
		s.Error(w, r, http.StatusNotFound, "Not found")
		return
	}

	// Build target path and verify it's a regular file
	target := filepath.Join(staticDir, rel)
	if fi, err := os.Stat(target); err != nil || fi.IsDir() {
		s.Error(w, r, http.StatusNotFound, "Not found")
		return
	}

	// Serve the file (Go's http.ServeFile handles content-type automatically)
	http.ServeFile(w, r, target)
}
