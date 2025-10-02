package web

import (
	"encoding/json"
	"net/http"
	"strings"

	"groupie-tracker/internal/data"
)

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
		searchResults = s.svc.SearchArtists(searchParams)
	}

	filterOptions := s.svc.GetArtistFilterOptions()

	// Generate all search suggestions for datalist
	allSuggestions := s.svc.GenerateAllSearchSuggestions()

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
	suggestions := s.svc.GenerateAllSearchSuggestions()
	matchingSuggestions := data.FilterSuggestionsOptimized(suggestions, query, maxSuggestions)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(matchingSuggestions)
}
