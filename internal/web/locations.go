package web

import (
	"fmt"
	"net/http"
	"strings"

	"groupie-tracker/internal/data"
)

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
