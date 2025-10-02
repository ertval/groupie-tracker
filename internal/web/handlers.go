package web

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"groupie-tracker/internal/data"
)

// ============================================================================
// HOME PAGE
// ============================================================================

// Home handles the home page.
func (s *Server) Home(w http.ResponseWriter, r *http.Request) {
	// Validate path using centralized utility
	if !s.validateExactPath(w, r, "/") {
		return
	}

	artists := s.store.Artists()
	stats := s.store.Stats()
	suggestions := s.store.GenerateAllSearchSuggestions()

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

	s.render(w, r, "home.tmpl", data)
}

// ============================================================================
// ARTISTS
// ============================================================================

// Artists handles the artists listing page.
func (s *Server) Artists(w http.ResponseWriter, r *http.Request) {
	// Validate path for both GET and POST using centralized utility
	if !s.validateExactPath(w, r, "/artists") {
		return
	}

	artists := s.store.Artists()
	filterOptions := s.store.GetArtistFilterOptions()
	suggestions := s.store.GenerateAllSearchSuggestions()
	var appliedFilters data.ArtistFilterParams
	totalArtists := len(artists)

	// If POST request, parse form data and apply filters
	if r.Method == http.MethodPost {
		if !s.parseFormOrError(w, r) {
			return
		}

		appliedFilters = parseArtistFilterParams(r)
		artists = s.store.FilterArtists(appliedFilters)
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

	s.render(w, r, "artists.tmpl", data)
}

// ArtistDetail handles individual artist pages.
func (s *Server) ArtistDetail(w http.ResponseWriter, r *http.Request) {
	// Validate path
	path := strings.TrimPrefix(r.URL.Path, "/artists/")
	if path == "" {
		s.NotFoundError(w, r, "")
		return
	}

	// Try slug first, then ID
	artist, found := s.store.ArtistBySlug(path)
	if !found {
		if id, err := strconv.Atoi(path); err == nil {
			artist, found = s.store.ArtistByID(id)
		}
		if !found {
			s.NotFoundError(w, r, "Artist not found")
			return
		}
	}

	// Get navigation artists using on-demand lookup
	prevArtist, nextArtist := s.store.GetAdjacentArtists(artist.ID)
	suggestions := s.store.GenerateAllSearchSuggestions()

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

	s.render(w, r, "artist_detail.tmpl", data)
}

// ============================================================================
// LOCATIONS
// ============================================================================

// Locations handles the locations listing page.
func (s *Server) Locations(w http.ResponseWriter, r *http.Request) {
	// Validate path for both GET and POST using centralized utility
	if !s.validateExactPath(w, r, "/locations") {
		return
	}

	locations := s.store.Locations()
	filterOptions := s.store.GetLocationFilterOptions()
	suggestions := s.store.GenerateAllSearchSuggestions()
	var appliedFilters data.LocationFilterParams
	totalLocations := len(locations)
	stats := s.store.Stats()

	// If POST request, parse form data and apply filters
	if r.Method == http.MethodPost {
		if !s.parseFormOrError(w, r) {
			return
		}

		appliedFilters = parseLocationFilterParams(r)
		locations = s.store.FilterLocations(appliedFilters)
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

	s.render(w, r, "locations.tmpl", data)
}

// LocationDetail handles individual location pages.
func (s *Server) LocationDetail(w http.ResponseWriter, r *http.Request) {
	slug := strings.TrimPrefix(r.URL.Path, "/locations/")
	if slug == "" {
		s.NotFoundError(w, r, "")
		return
	}

	location, found := s.store.LocationBySlug(slug)
	if !found {
		s.NotFoundError(w, r, "Location not found")
		return
	}

	suggestions := s.store.GenerateAllSearchSuggestions()

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

	s.render(w, r, "location_detail.tmpl", data)
}

// ============================================================================
// SEARCH
// ============================================================================

// Search handles search requests for artists.
func (s *Server) Search(w http.ResponseWriter, r *http.Request) {
	// Validate path using centralized utility
	if !s.validateExactPath(w, r, "/search") {
		return
	}

	var searchQuery string
	var appliedFilters data.ArtistFilterParams
	var searchResults data.SearchResult

	// Handle search submission
	if r.Method == http.MethodPost {
		if !s.parseFormOrError(w, r) {
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
		searchResults = s.store.SearchArtists(searchParams)
	}

	filterOptions := s.store.GetArtistFilterOptions()

	// Generate all search suggestions for datalist
	allSuggestions := s.store.GenerateAllSearchSuggestions()

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

	s.render(w, r, "search.tmpl", data)
}

// SuggestionsAPI provides search suggestions for autocomplete functionality.
func (s *Server) SuggestionsAPI(w http.ResponseWriter, r *http.Request) {
	query := strings.TrimSpace(r.URL.Query().Get("q"))

	// Use optimized filtering with reasonable limits
	const maxSuggestions = 15 // Limit to avoid overwhelming the UI
	matchingSuggestions := s.store.FilterSearchSuggestions(query, maxSuggestions)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(matchingSuggestions)
}

// ============================================================================
// HEALTH CHECK
// ============================================================================

// Health provides a health check endpoint.
func (s *Server) Health(w http.ResponseWriter, r *http.Request) {
	response := map[string]any{
		"status":    "healthy",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"stats":     s.store.Stats(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// ============================================================================
// DEVELOPER TOOLS
// ============================================================================

// DevIndex renders a small developer page with quick links.
func (s *Server) DevIndex(w http.ResponseWriter, r *http.Request) {
	links := []struct{ Href, Text string }{
		{"/dev/panic", "Trigger Panic (/dev/panic)"},
		{"/dev/404", "Simulate 404 (/dev/404)"},
		{"/dev/500", "Simulate 500 (/dev/500)"},
		{"/dev/tmpl-error", "Simulate Template Error (/dev/tmpl-error)"},
		{"/health", "Health Check (/health)"},
	}

	suggestions := s.store.GenerateAllSearchSuggestions()

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

	s.render(w, r, "dev.tmpl", data)
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
