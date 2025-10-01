package web

import (
	"net/http"
	"sort"
	"strconv"
	"strings"

	"groupie-tracker/internal/data"
)

// Common template data structure
type BasePage struct {
	Title       string
	ExtraCSS    string
	ExtraJS     string
	Suggestions []Suggestion
}

type Suggestion struct {
	Text        string
	Description string
}

// Home handles the home page.
func (s *Server) Home(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		s.errorHandler(w, r, http.StatusNotFound, "Page not found")
		return
	}

	artists := s.store.Artists()
	stats := s.store.Stats()

	// Get 8 random artists for homepage
	if len(artists) > 8 {
		artists = artists[:8] // Simple approach - take first 8
	}

	pageData := struct {
		BasePage
		Artists []data.Artist
		Stats   data.Stats
	}{
		BasePage: BasePage{
			Title:       "Groupie Tracker",
			ExtraCSS:    "home.css",
			ExtraJS:     "",
			Suggestions: []Suggestion{},
		},
		Artists: artists,
		Stats:   stats,
	}

	s.render(w, "home.tmpl", pageData)
}

// Artists handles the artists listing page.
func (s *Server) Artists(w http.ResponseWriter, r *http.Request) {
	artists := s.store.Artists()
	filterOptions := s.store.GetArtistFilterOptions()
	var appliedFilters data.ArtistFilterParams
	isFiltered := false

	// Handle POST requests (filters)
	if r.Method == http.MethodPost {
		if err := r.ParseForm(); err != nil {
			s.errorHandler(w, r, http.StatusBadRequest, "Invalid form data")
			return
		}

		appliedFilters = s.parseArtistFilters(r)
		artists = s.store.FilterArtists(appliedFilters)
		isFiltered = true
	}

	// Sort by concert count
	sort.Slice(artists, func(i, j int) bool {
		return artists[i].ConcertCount > artists[j].ConcertCount
	})

	data := struct {
		BasePage
		Artists        []data.Artist
		FilterOptions  data.ArtistFilterOptions
		AppliedFilters data.ArtistFilterParams
		IsFiltered     bool
	}{
		BasePage: BasePage{
			Title:       "Artists",
			ExtraCSS:    "artists.css",
			ExtraJS:     "",
			Suggestions: []Suggestion{},
		},
		Artists:        artists,
		FilterOptions:  filterOptions,
		AppliedFilters: appliedFilters,
		IsFiltered:     isFiltered,
	}

	s.render(w, "artists.tmpl", data)
}

// ArtistDetail handles individual artist pages.
func (s *Server) ArtistDetail(w http.ResponseWriter, r *http.Request) {
	slug := strings.TrimPrefix(r.URL.Path, "/artists/")
	if slug == "" {
		s.errorHandler(w, r, http.StatusNotFound, "Artist not found")
		return
	}

	artist, exists := s.store.ArtistBySlug(slug)
	if !exists {
		s.errorHandler(w, r, http.StatusNotFound, "Artist not found")
		return
	}

	data := struct {
		BasePage
		Artist data.Artist
	}{
		BasePage: BasePage{
			Title:       artist.Name,
			ExtraCSS:    "artist_detail.css",
			ExtraJS:     "",
			Suggestions: []Suggestion{},
		},
		Artist: artist,
	}

	s.render(w, "artist_detail.tmpl", data)
}

// Locations handles the locations listing page.
func (s *Server) Locations(w http.ResponseWriter, r *http.Request) {
	locations := s.store.Locations()
	filterOptions := s.store.GetLocationFilterOptions()
	var appliedFilters data.LocationFilterParams
	isFiltered := false

	// Handle POST requests (filters)
	if r.Method == http.MethodPost {
		if err := r.ParseForm(); err != nil {
			s.errorHandler(w, r, http.StatusBadRequest, "Invalid form data")
			return
		}

		appliedFilters = s.parseLocationFilters(r)
		locations = s.store.FilterLocations(appliedFilters)
		isFiltered = true
	}

	data := struct {
		BasePage
		Locations      []data.Location
		FilterOptions  data.LocationFilterOptions
		AppliedFilters data.LocationFilterParams
		IsFiltered     bool
	}{
		BasePage: BasePage{
			Title:       "Locations",
			ExtraCSS:    "locations.css",
			ExtraJS:     "",
			Suggestions: []Suggestion{},
		},
		Locations:      locations,
		FilterOptions:  filterOptions,
		AppliedFilters: appliedFilters,
		IsFiltered:     isFiltered,
	}

	s.render(w, "locations.tmpl", data)
}

// LocationDetail handles individual location pages.
func (s *Server) LocationDetail(w http.ResponseWriter, r *http.Request) {
	slug := strings.TrimPrefix(r.URL.Path, "/locations/")
	if slug == "" {
		s.errorHandler(w, r, http.StatusNotFound, "Location not found")
		return
	}

	location, exists := s.store.LocationBySlug(slug)
	if !exists {
		s.errorHandler(w, r, http.StatusNotFound, "Location not found")
		return
	}

	data := struct {
		BasePage
		Location data.Location
	}{
		BasePage: BasePage{
			Title:       location.Name,
			ExtraCSS:    "location_detail.css",
			ExtraJS:     "",
			Suggestions: []Suggestion{},
		},
		Location: location,
	}

	s.render(w, "location_detail.tmpl", data)
}

// Search handles search requests.
func (s *Server) Search(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	results := data.SearchResults{}

	if query != "" {
		params := data.SearchParams{
			Query: query,
			Type:  "all",
			Limit: 50,
		}
		results = s.store.Search(params)
	}

	data := struct {
		BasePage
		Query          string
		Results        data.SearchResults
		IsSearch       bool
		FilterOptions  data.ArtistFilterOptions
		AppliedFilters data.ArtistFilterParams
	}{
		BasePage: BasePage{
			Title:       "Search",
			ExtraCSS:    "search.css",
			ExtraJS:     "",
			Suggestions: []Suggestion{},
		},
		Query:          query,
		Results:        results,
		IsSearch:       query != "",
		FilterOptions:  s.store.GetArtistFilterOptions(),
		AppliedFilters: data.ArtistFilterParams{},
	}

	s.render(w, "search.tmpl", data)
}

// Health handles health check requests.
func (s *Server) Health(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"healthy"}`))
}

// --- Helper methods ---

func (s *Server) parseArtistFilters(r *http.Request) data.ArtistFilterParams {
	var filters data.ArtistFilterParams

	// Parse creation year range
	if yearFrom := r.FormValue("creationYearFrom"); yearFrom != "" {
		if year, err := strconv.Atoi(yearFrom); err == nil {
			filters.CreationYearFrom = &year
		}
	}
	if yearTo := r.FormValue("creationYearTo"); yearTo != "" {
		if year, err := strconv.Atoi(yearTo); err == nil {
			filters.CreationYearTo = &year
		}
	}

	// Parse member counts
	if memberCounts := r.Form["memberCounts"]; len(memberCounts) > 0 {
		for _, countStr := range memberCounts {
			if count, err := strconv.Atoi(countStr); err == nil {
				filters.MemberCounts = append(filters.MemberCounts, count)
			}
		}
	}

	// Parse countries
	filters.Countries = r.Form["countries"]

	return filters
}

func (s *Server) parseLocationFilters(r *http.Request) data.LocationFilterParams {
	var filters data.LocationFilterParams

	// Parse year range
	if yearFrom := r.FormValue("yearFrom"); yearFrom != "" {
		if year, err := strconv.Atoi(yearFrom); err == nil {
			filters.YearFrom = &year
		}
	}
	if yearTo := r.FormValue("yearTo"); yearTo != "" {
		if year, err := strconv.Atoi(yearTo); err == nil {
			filters.YearTo = &year
		}
	}

	// Parse artist counts
	if artistCounts := r.Form["artistCounts"]; len(artistCounts) > 0 {
		for _, countStr := range artistCounts {
			if count, err := strconv.Atoi(countStr); err == nil {
				filters.ArtistCounts = append(filters.ArtistCounts, count)
			}
		}
	}

	return filters
}

func (s *Server) errorHandler(w http.ResponseWriter, r *http.Request, statusCode int, message string) {
	w.WriteHeader(statusCode)

	data := struct {
		BasePage
		Code         int
		Message      string
		RequestedURL string
		Timestamp    string
	}{
		BasePage: BasePage{
			Title:       "Error",
			ExtraCSS:    "errors.css",
			ExtraJS:     "",
			Suggestions: []Suggestion{},
		},
		Code:         statusCode,
		Message:      message,
		RequestedURL: r.URL.Path,
		Timestamp:    "2025-09-30 18:25:22", // Simple timestamp
	}

	s.render(w, "error.tmpl", data)
}
