package data

import (
	"context"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"groupie-tracker/internal/api"
	"groupie-tracker/internal/config"
)

// Store provides centralized data management with pre-computed indexes.
type Store struct {
	// Core data collections
	artists   []Artist
	locations []Location

	// Fast lookup indexes
	artistsByID     map[int]*Artist
	artistsBySlug   map[string]*Artist
	locationsBySlug map[string]*Location

	// Pre-computed for UI
	suggestions []SearchSuggestion
	stats       Stats

	// API client
	apiClient *api.Client
}

// NewStore creates a new data store instance.
func NewStore() *Store {
	return &Store{
		artistsByID:     make(map[int]*Artist),
		artistsBySlug:   make(map[string]*Artist),
		locationsBySlug: make(map[string]*Location),
		apiClient:       api.NewClient(config.APIBaseURL, config.APIRequestTimeout),
	}
}

// LoadData loads and processes all data from the external API.
func (s *Store) LoadData(ctx context.Context) error {
	// Fetch raw data from API
	apiArtists, err := s.apiClient.FetchArtists(ctx)
	if err != nil {
		return fmt.Errorf("failed to fetch artists: %w", err)
	}

	apiRelations, err := s.apiClient.FetchRelations(ctx)
	if err != nil {
		return fmt.Errorf("failed to fetch relations: %w", err)
	}

	// Transform API data to domain models
	s.artists = s.transformArtists(apiArtists, apiRelations.Index)

	// Build indexes
	s.buildIndexes()

	// Generate locations from artist data
	s.buildLocations()

	// Pre-compute suggestions and stats
	s.buildSuggestions()
	s.buildStats()

	return nil
}

// Artists returns all artists sorted by name.
func (s *Store) Artists() []Artist {
	return s.artists
}

// ArtistBySlug returns an artist by slug.
func (s *Store) ArtistBySlug(slug string) (Artist, bool) {
	artist, exists := s.artistsBySlug[slug]
	if !exists {
		return Artist{}, false
	}
	return *artist, true
}

// Locations returns all locations sorted by concert count.
func (s *Store) Locations() []Location {
	return s.locations
}

// LocationBySlug returns a location by slug.
func (s *Store) LocationBySlug(slug string) (Location, bool) {
	location, exists := s.locationsBySlug[slug]
	if !exists {
		return Location{}, false
	}
	return *location, true
}

// SearchSuggestions returns all search suggestions.
func (s *Store) SearchSuggestions() []SearchSuggestion {
	return s.suggestions
}

// Stats returns application statistics.
func (s *Store) Stats() Stats {
	return s.stats
}

// FilterArtists applies filter criteria to the artist collection.
func (s *Store) FilterArtists(params ArtistFilterParams) []Artist {
	filtered := make([]Artist, 0, len(s.artists))

	for _, artist := range s.artists {
		if s.matchesArtistFilter(artist, params) {
			filtered = append(filtered, artist)
		}
	}

	return filtered
}

// FilterLocations applies filter criteria to the location collection.
func (s *Store) FilterLocations(params LocationFilterParams) []Location {
	filtered := make([]Location, 0, len(s.locations))

	for _, location := range s.locations {
		if s.matchesLocationFilter(location, params) {
			filtered = append(filtered, location)
		}
	}

	return filtered
}

// GetArtistFilterOptions returns available filter options for artists.
func (s *Store) GetArtistFilterOptions() ArtistFilterOptions {
	minYear, maxYear := 9999, 0
	memberCounts := make(map[int]bool)
	countries := make(map[string]bool)

	for _, artist := range s.artists {
		if artist.CreationYear < minYear {
			minYear = artist.CreationYear
		}
		if artist.CreationYear > maxYear {
			maxYear = artist.CreationYear
		}

		memberCounts[len(artist.Members)] = true

		for _, country := range artist.Countries {
			countries[country] = true
		}
	}

	// Convert maps to slices
	memberCountSlice := make([]int, 0, len(memberCounts))
	for count := range memberCounts {
		memberCountSlice = append(memberCountSlice, count)
	}

	countrySlice := make([]string, 0, len(countries))
	for country := range countries {
		countrySlice = append(countrySlice, country)
	}

	return ArtistFilterOptions{
		CreationYearMin: minYear,
		CreationYearMax: maxYear,
		MemberCounts:    memberCountSlice,
		Countries:       countrySlice,
	}
}

// GetLocationFilterOptions returns available filter options for locations.
func (s *Store) GetLocationFilterOptions() LocationFilterOptions {
	minYear, maxYear := 9999, 0
	artistCounts := make(map[int]bool)

	for _, location := range s.locations {
		if location.YearRange[0] > 0 && location.YearRange[0] < minYear {
			minYear = location.YearRange[0]
		}
		if location.YearRange[1] > maxYear {
			maxYear = location.YearRange[1]
		}

		artistCounts[len(location.Artists)] = true
	}

	artistCountSlice := make([]int, 0, len(artistCounts))
	for count := range artistCounts {
		artistCountSlice = append(artistCountSlice, count)
	}

	return LocationFilterOptions{
		YearMin:      minYear,
		YearMax:      maxYear,
		ArtistCounts: artistCountSlice,
	}
}

// Search performs a comprehensive search across artists and locations.
func (s *Store) Search(params SearchParams) SearchResults {
	query := strings.ToLower(strings.TrimSpace(params.Query))
	if query == "" {
		return SearchResults{}
	}

	var artists []Artist
	var locations []Location

	// Search artists
	if params.Type == "all" || params.Type == "artists" {
		artists = s.searchArtists(query, params.Limit)
	}

	// Search locations
	if params.Type == "all" || params.Type == "locations" {
		locations = s.searchLocations(query, params.Limit)
	}

	return SearchResults{
		Artists:   artists,
		Locations: locations,
		Total:     len(artists) + len(locations),
	}
}

// --- Private helper methods ---

func (s *Store) transformArtists(apiArtists []api.APIArtist, relations []api.APIRelationIndex) []Artist {
	relationMap := make(map[int]api.APIRelationIndex)
	for _, rel := range relations {
		relationMap[rel.ID] = rel
	}

	artists := make([]Artist, 0, len(apiArtists))
	for _, apiArtist := range apiArtists {
		artist := Artist{
			ID:           apiArtist.ID,
			Name:         apiArtist.Name,
			Slug:         createSlug(apiArtist.Name),
			Members:      apiArtist.Members,
			CreationYear: apiArtist.CreationDate,
			FirstAlbum:   apiArtist.FirstAlbum,
			Image:        apiArtist.Image,
		}

		// Add concerts and compute countries
		if rel, exists := relationMap[artist.ID]; exists {
			artist.Concerts = s.buildConcerts(rel.DatesLocations)
			artist.Countries = s.extractCountries(rel.DatesLocations)
			artist.ConcertCount = len(artist.Concerts)
		}

		artists = append(artists, artist)
	}

	return artists
}

func (s *Store) buildConcerts(datesLocations map[string][]string) []Concert {
	var concerts []Concert
	for location, dates := range datesLocations {
		normalizedLocation := normalizeLocation(location)
		for _, date := range dates {
			concerts = append(concerts, Concert{
				Date:     date,
				Location: normalizedLocation,
			})
		}
	}
	return concerts
}

func (s *Store) extractCountries(datesLocations map[string][]string) []string {
	countrySet := make(map[string]bool)
	for location := range datesLocations {
		parts := strings.Split(location, "-")
		if len(parts) >= 2 {
			country := strings.ToUpper(parts[len(parts)-1])
			countrySet[country] = true
		}
	}

	countries := make([]string, 0, len(countrySet))
	for country := range countrySet {
		countries = append(countries, country)
	}
	sort.Strings(countries)
	return countries
}

func (s *Store) buildIndexes() {
	for i := range s.artists {
		artist := &s.artists[i]
		s.artistsByID[artist.ID] = artist
		s.artistsBySlug[artist.Slug] = artist
	}
}

func (s *Store) buildLocations() {
	locationMap := make(map[string]*Location)

	for _, artist := range s.artists {
		for _, concert := range artist.Concerts {
			slug := createSlug(concert.Location)

			if location, exists := locationMap[slug]; exists {
				// Add artist to existing location
				location.ConcertCount++
				s.addArtistToLocation(location, artist)
			} else {
				// Create new location
				location := &Location{
					Name:         concert.Location,
					Slug:         slug,
					Artists:      []ArtistSummary{{Name: artist.Name, Slug: artist.Slug, ConcertCount: 1}},
					ConcertCount: 1,
					YearRange:    s.extractYearFromDate(concert.Date),
				}
				locationMap[slug] = location
			}
		}
	}

	// Convert map to slice and sort
	s.locations = make([]Location, 0, len(locationMap))
	for _, location := range locationMap {
		s.locations = append(s.locations, *location)
		s.locationsBySlug[location.Slug] = location
	}

	sort.Slice(s.locations, func(i, j int) bool {
		return s.locations[i].ConcertCount > s.locations[j].ConcertCount
	})
}

func (s *Store) addArtistToLocation(location *Location, artist Artist) {
	// Check if artist already exists in location
	for i, artistSummary := range location.Artists {
		if artistSummary.Slug == artist.Slug {
			location.Artists[i].ConcertCount++
			return
		}
	}

	// Add new artist to location
	location.Artists = append(location.Artists, ArtistSummary{
		Name:         artist.Name,
		Slug:         artist.Slug,
		ConcertCount: 1,
	})
}

func (s *Store) extractYearFromDate(dateStr string) [2]int {
	// Extract year from date string (format varies)
	parts := strings.Split(dateStr, "-")
	if len(parts) >= 3 {
		// Assuming DD-MM-YYYY or MM-DD-YYYY format
		if year := parseInt(parts[2]); year > 0 {
			return [2]int{year, year}
		}
	}
	return [2]int{0, 0}
}

func (s *Store) buildSuggestions() {
	suggestions := make([]SearchSuggestion, 0, len(s.artists)+len(s.locations))

	// Add artist suggestions
	for _, artist := range s.artists {
		suggestions = append(suggestions, SearchSuggestion{
			Text:     artist.Name,
			Type:     "artist",
			Category: "Artists",
		})
	}

	// Add location suggestions
	for _, location := range s.locations {
		suggestions = append(suggestions, SearchSuggestion{
			Text:     location.Name,
			Type:     "location",
			Category: "Locations",
		})
	}

	s.suggestions = suggestions
}

func (s *Store) buildStats() {
	countrySet := make(map[string]bool)
	totalMembers := 0
	totalConcerts := 0

	for _, artist := range s.artists {
		totalMembers += len(artist.Members)
		totalConcerts += artist.ConcertCount
		for _, country := range artist.Countries {
			countrySet[country] = true
		}
	}

	s.stats = Stats{
		TotalArtists:   len(s.artists),
		TotalLocations: len(s.locations),
		TotalMembers:   totalMembers,
		TotalConcerts:  totalConcerts,
		TotalCountries: len(countrySet),
	}
}

func parseInt(s string) int {
	// Simple integer parsing - extend as needed
	result := 0
	for _, r := range s {
		if r >= '0' && r <= '9' {
			result = result*10 + int(r-'0')
		} else {
			break
		}
	}
	return result
}

// --- Private helper methods ---

func (s *Store) matchesArtistFilter(artist Artist, params ArtistFilterParams) bool {
	// Creation year filter
	if params.CreationYearFrom != nil && artist.CreationYear < *params.CreationYearFrom {
		return false
	}
	if params.CreationYearTo != nil && artist.CreationYear > *params.CreationYearTo {
		return false
	}

	// Member count filter
	if len(params.MemberCounts) > 0 {
		memberCount := len(artist.Members)
		found := false
		for _, count := range params.MemberCounts {
			if memberCount == count {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// Countries filter
	if len(params.Countries) > 0 {
		found := false
		for _, filterCountry := range params.Countries {
			for _, artistCountry := range artist.Countries {
				if artistCountry == filterCountry {
					found = true
					break
				}
			}
			if found {
				break
			}
		}
		if !found {
			return false
		}
	}

	return true
}

func (s *Store) matchesLocationFilter(location Location, params LocationFilterParams) bool {
	// Year range filter
	if params.YearFrom != nil && location.YearRange[1] < *params.YearFrom {
		return false
	}
	if params.YearTo != nil && location.YearRange[0] > *params.YearTo {
		return false
	}

	// Artist count filter
	if len(params.ArtistCounts) > 0 {
		artistCount := len(location.Artists)
		found := false
		for _, count := range params.ArtistCounts {
			if artistCount == count {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	return true
}

// searchArtists searches for artists matching the query.
func (s *Store) searchArtists(query string, limit int) []Artist {
	var results []Artist

	for _, artist := range s.artists {
		if s.artistMatches(artist, query) {
			results = append(results, artist)
			if limit > 0 && len(results) >= limit {
				break
			}
		}
	}

	return results
}

// searchLocations searches for locations matching the query.
func (s *Store) searchLocations(query string, limit int) []Location {
	var results []Location

	for _, location := range s.locations {
		if s.locationMatches(location, query) {
			results = append(results, location)
			if limit > 0 && len(results) >= limit {
				break
			}
		}
	}

	return results
}

// artistMatches checks if an artist matches the search query.
func (s *Store) artistMatches(artist Artist, query string) bool {
	// Name match
	if strings.Contains(strings.ToLower(artist.Name), query) {
		return true
	}

	// Member match
	for _, member := range artist.Members {
		if strings.Contains(strings.ToLower(member), query) {
			return true
		}
	}

	// Creation year match
	if strings.Contains(strconv.Itoa(artist.CreationYear), query) {
		return true
	}

	// First album match
	if strings.Contains(strings.ToLower(artist.FirstAlbum), query) {
		return true
	}

	// Country match
	for _, country := range artist.Countries {
		if strings.Contains(strings.ToLower(country), query) {
			return true
		}
	}

	return false
}

// locationMatches checks if a location matches the search query.
func (s *Store) locationMatches(location Location, query string) bool {
	return strings.Contains(strings.ToLower(location.Name), query)
}

// --- Utility functions ---

func createSlug(name string) string {
	reg := regexp.MustCompile(`[^a-z0-9]+`)
	slug := reg.ReplaceAllString(strings.ToLower(name), "-")
	return strings.Trim(slug, "-")
}

func normalizeLocation(location string) string {
	return strings.ToLower(strings.ReplaceAll(strings.TrimSpace(location), "_", "-"))
}
