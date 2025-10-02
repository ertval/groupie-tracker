package data

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"

	"groupie-tracker/internal/api"
)

// Store holds the immutable in-memory data for the application.
// After Load() completes, all fields are read-only and thread-safe.
type Store struct {
	// API client for external data fetching
	apiClient *api.Client

	// Configuration
	withCache bool

	// Cache status
	cacheEnabled bool

	// Core data collections (immutable after Load)
	artists         []Artist
	artistsByID     map[int]Artist
	artistsBySlug   map[string]Artist
	artistPositions map[int]int
	locations       []Location
	locationsBySlug map[string]Location
	appStats        AppStats
	suggestions     []SearchSuggestion
	artistFilters   ArtistFilterOptions
	locationFilters LocationFilterOptions

	// Search cache (LRU-style)
	searchCacheMu   sync.Mutex
	searchCache     map[string][]Artist
	searchOrder     []string
	searchCacheSize int

	// Ensure Load is called only once
	loadOnce sync.Once
	loadErr  error
}

// NewStore creates a new Store instance with the provided API client.
func NewStore(apiClient *api.Client, withCache bool) *Store {
	return &Store{
		apiClient:       apiClient,
		withCache:       withCache,
		searchCache:     make(map[string][]Artist, 50),
		searchOrder:     make([]string, 0, 50),
		searchCacheSize: 50,
	}
}

// Load fetches and processes all data from the API.
// This method is thread-safe and will only execute once.
func (s *Store) Load(ctx context.Context) error {
	s.loadOnce.Do(func() {
		s.loadErr = s.loadData(ctx)
	})
	return s.loadErr
}

// loadData performs the actual data loading (called once by Load)
func (s *Store) loadData(ctx context.Context) error {
	// Fetch raw data from API concurrently using goroutines
	artistsCh := make(chan struct {
		data []api.Artist
		err  error
	}, 1)
	relationsCh := make(chan struct {
		data api.Relation
		err  error
	}, 1)

	// Fetch artists in parallel
	go func() {
		data, err := s.apiClient.FetchArtists(ctx)
		artistsCh <- struct {
			data []api.Artist
			err  error
		}{data, err}
	}()

	// Fetch relations in parallel
	go func() {
		data, err := s.apiClient.FetchRelations(ctx)
		relationsCh <- struct {
			data api.Relation
			err  error
		}{data, err}
	}()

	// Wait for both fetches to complete and check for errors
	artistsResult := <-artistsCh
	relationsResult := <-relationsCh

	if artistsResult.err != nil {
		return fmt.Errorf("failed to fetch artists: %w", artistsResult.err)
	}
	if relationsResult.err != nil {
		return fmt.Errorf("failed to fetch relations: %w", relationsResult.err)
	}

	artists := s.processArtists(artistsResult.data, relationsResult.data)

	var cachedImages, downloadedImages int
	if s.withCache {
		var cacheEnabled bool
		cacheEnabled, cachedImages, downloadedImages = s.cacheImages(artists)
		s.cacheEnabled = cacheEnabled
	} else {
		s.cacheEnabled = false
	}

	s.artists = artists

	var (
		artistsByID     map[int]Artist
		artistsBySlug   map[string]Artist
		artistPositions map[int]int
		locations       []Location
		locationsBySlug map[string]Location
		artistFilters   ArtistFilterOptions
		locationFilters LocationFilterOptions
		suggestions     []SearchSuggestion
	)

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		artistsByID, artistsBySlug, artistPositions = s.createArtistIndexes(artists)
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		locations, locationsBySlug = s.createLocationsData(artists)
		locationFilters = s.calculateLocationFilterOptions(locations)
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		artistFilters = s.calculateArtistFilterOptions(artists)
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		suggestions = s.generateSearchSuggestions(artists)
	}()

	wg.Wait()

	s.artistsByID = artistsByID
	s.artistsBySlug = artistsBySlug
	s.artistPositions = artistPositions
	s.locations = locations
	s.locationsBySlug = locationsBySlug
	s.artistFilters = artistFilters
	s.locationFilters = locationFilters
	s.suggestions = suggestions
	s.appStats = s.calculateStats(artists, locations, cachedImages, downloadedImages)

	return nil
}

// Read-only accessors

// Artists returns all artists sorted by name.
func (s *Store) Artists() []Artist {
	return s.artists
}

// ArtistByID returns an artist by ID, or false if not found.
func (s *Store) ArtistByID(id int) (Artist, bool) {
	artist, ok := s.artistsByID[id]
	return artist, ok
}

// ArtistBySlug returns an artist by URL slug, or false if not found.
func (s *Store) ArtistBySlug(slug string) (Artist, bool) {
	artist, ok := s.artistsBySlug[slug]
	return artist, ok
}

// ArtistPosition returns the index of the artist within the sorted slice.
func (s *Store) ArtistPosition(id int) (int, bool) {
	index, ok := s.artistPositions[id]
	return index, ok
}

// Locations returns all locations sorted by concert count.
func (s *Store) Locations() []Location {
	return s.locations
}

// LocationBySlug returns a location by URL slug, or false if not found.
func (s *Store) LocationBySlug(slug string) (Location, bool) {
	location, ok := s.locationsBySlug[slug]
	return location, ok
}

// Stats returns application statistics.
func (s *Store) Stats() AppStats {
	return s.appStats
}

// CacheEnabled returns whether image caching is enabled and functional.
func (s *Store) CacheEnabled() bool {
	return s.cacheEnabled
}

// Suggestions returns the precomputed search suggestions for autocomplete.
func (s *Store) Suggestions() []SearchSuggestion {
	return s.suggestions
}

// ArtistFilterOptions returns the precomputed artist filter metadata.
func (s *Store) ArtistFilterOptions() ArtistFilterOptions {
	return s.artistFilters
}

// GetArtistFilterOptions returns the precomputed artist filter metadata.
func (s *Store) GetArtistFilterOptions() ArtistFilterOptions {
	return s.artistFilters
}

// LocationFilterOptions returns the precomputed location filter metadata.
func (s *Store) LocationFilterOptions() LocationFilterOptions {
	return s.locationFilters
}

// GetLocationFilterOptions returns the precomputed location filter metadata.
func (s *Store) GetLocationFilterOptions() LocationFilterOptions {
	return s.locationFilters
}

// ============================================================================
// BUSINESS LOGIC - Filtering
// ============================================================================

// FilterArtists filters artists based on criteria like creation date, album date, location, and member count.
func (s *Store) FilterArtists(criteria ArtistFilterParams) []Artist {
	artists := s.Artists()
	if len(artists) == 0 {
		return nil
	}

	var filtered []Artist
	for _, artist := range artists {
		if matchesArtistFilters(artist, criteria) {
			filtered = append(filtered, artist)
		}
	}

	return filtered
}

// FilterLocations filters locations based on concert count, artist count, year range, and country.
func (s *Store) FilterLocations(params LocationFilterParams) []Location {
	locations := s.Locations()
	if len(locations) == 0 {
		return nil
	}

	var filtered []Location
	for _, location := range locations {
		if matchesLocationFilters(location, params) {
			filtered = append(filtered, location)
		}
	}

	return filtered
}

// matchesArtistFilters checks if an artist matches all specified filter criteria.
func matchesArtistFilters(artist Artist, params ArtistFilterParams) bool {
	if params.CreationYearFrom != nil && artist.CreationYear < *params.CreationYearFrom {
		return false
	}
	if params.CreationYearTo != nil && artist.CreationYear > *params.CreationYearTo {
		return false
	}

	if params.FirstAlbumYearFrom != nil || params.FirstAlbumYearTo != nil {
		albumYear := artist.FirstAlbumYear
		if albumYear > 0 {
			if params.FirstAlbumYearFrom != nil && albumYear < *params.FirstAlbumYearFrom {
				return false
			}
			if params.FirstAlbumYearTo != nil && albumYear > *params.FirstAlbumYearTo {
				return false
			}
		}
	}

	if len(params.MemberCounts) > 0 {
		memberCount := artist.MemberCount
		found := false
		for _, allowedCount := range params.MemberCounts {
			if memberCount == allowedCount {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	if len(params.Countries) > 0 {
		allowed := make(map[string]struct{}, len(params.Countries))
		for _, country := range params.Countries {
			allowed[country] = struct{}{}
		}

		hasMatchingCountry := false
		for _, country := range artist.Countries {
			if _, ok := allowed[country]; ok {
				hasMatchingCountry = true
				break
			}
		}
		if !hasMatchingCountry {
			return false
		}
	}

	return true
}

// matchesLocationFilters checks if a location matches all specified filter criteria.
func matchesLocationFilters(location Location, params LocationFilterParams) bool {
	if params.ConcertCountFrom != nil && location.TotalConcerts < *params.ConcertCountFrom {
		return false
	}
	if params.ConcertCountTo != nil && location.TotalConcerts > *params.ConcertCountTo {
		return false
	}

	if params.ArtistCountFrom != nil && location.ArtistCount < *params.ArtistCountFrom {
		return false
	}
	if params.ArtistCountTo != nil && location.ArtistCount > *params.ArtistCountTo {
		return false
	}

	if params.ConcertYearFrom != nil && location.LatestYear < *params.ConcertYearFrom {
		return false
	}
	if params.ConcertYearTo != nil && location.EarliestYear > *params.ConcertYearTo {
		return false
	}

	if len(params.Countries) > 0 {
		locationCountry := location.Country
		for _, allowedCountry := range params.Countries {
			if locationCountry == allowedCountry {
				return true
			}
		}
		return false
	}

	return true
}

// ============================================================================
// BUSINESS LOGIC - Search
// ============================================================================

// SearchArtists performs search across artist data with optional filtering.
func (s *Store) SearchArtists(params SearchParams) SearchResult {
	artists := s.Artists()
	normalizedQuery := normalizeSearchQuery(params.Query)
	filtersEmpty := isEmptyFilter(params.Filters)
	useCache := normalizedQuery != "" && filtersEmpty

	if useCache {
		if cached, ok := s.getCachedSearchResults(normalizedQuery); ok {
			return SearchResult{
				Artists:      cached,
				Query:        params.Query,
				TotalResults: len(cached),
			}
		}
	}

	var matchingArtists []Artist

	if normalizedQuery == "" {
		matchingArtists = artists
	} else {
		for _, artist := range artists {
			if matchesSearchQuery(artist, normalizedQuery) {
				matchingArtists = append(matchingArtists, artist)
			}
		}
	}

	if !filtersEmpty {
		var filtered []Artist
		for _, artist := range matchingArtists {
			if matchesArtistFilters(artist, params.Filters) {
				filtered = append(filtered, artist)
			}
		}
		matchingArtists = filtered
	}

	if useCache {
		s.setCachedSearchResults(normalizedQuery, matchingArtists)
	}

	return SearchResult{
		Artists:      matchingArtists,
		Query:        params.Query,
		TotalResults: len(matchingArtists),
	}
}

// GenerateAllSearchSuggestions returns the precomputed suggestion cache.
func (s *Store) GenerateAllSearchSuggestions() []SearchSuggestion {
	return s.Suggestions()
}

// FilterSearchSuggestions returns suggestions matching the query ordered by relevance.
func (s *Store) FilterSearchSuggestions(query string, maxResults int) []SearchSuggestion {
	suggestions := s.Suggestions()
	return filterSearchSuggestions(suggestions, query, maxResults)
}

// GetAdjacentArtists finds the previous and next artists in alphabetical order.
func (s *Store) GetAdjacentArtists(currentID int) (prev, next *Artist) {
	index, ok := s.ArtistPosition(currentID)
	if !ok {
		return nil, nil
	}

	artists := s.Artists()
	if len(artists) == 0 {
		return nil, nil
	}

	if index > 0 {
		prev = &artists[index-1]
	}

	if index < len(artists)-1 {
		next = &artists[index+1]
	}

	return prev, next
}

// ============================================================================
// SEARCH HELPERS
// ============================================================================

func normalizeSearchQuery(query string) string {
	return strings.ToLower(strings.TrimSpace(query))
}

func filterSearchSuggestions(suggestions []SearchSuggestion, query string, maxResults int) []SearchSuggestion {
	normalizedQuery := normalizeSearchQuery(query)
	if normalizedQuery == "" || len(suggestions) == 0 {
		return []SearchSuggestion{}
	}

	if maxResults <= 0 {
		maxResults = 20
	}

	var exactMatches []SearchSuggestion
	var prefixMatches []SearchSuggestion
	var containsMatches []SearchSuggestion

	totalFound := 0

	for _, suggestion := range suggestions {
		if totalFound >= maxResults {
			break
		}

		normalizedText := normalizeSearchQuery(suggestion.Text)

		switch {
		case normalizedText == normalizedQuery:
			exactMatches = append(exactMatches, suggestion)
			totalFound++
		case strings.HasPrefix(normalizedText, normalizedQuery):
			prefixMatches = append(prefixMatches, suggestion)
			totalFound++
		case strings.Contains(normalizedText, normalizedQuery):
			containsMatches = append(containsMatches, suggestion)
			totalFound++
		}
	}

	results := make([]SearchSuggestion, 0, len(exactMatches)+len(prefixMatches)+len(containsMatches))
	results = append(results, exactMatches...)
	results = append(results, prefixMatches...)
	results = append(results, containsMatches...)

	if len(results) > maxResults {
		results = results[:maxResults]
	}

	return results
}

func matchesSearchQuery(artist Artist, normalizedQuery string) bool {
	if normalizedQuery == "" {
		return true
	}

	if strings.Contains(strings.ToLower(artist.Name), normalizedQuery) {
		return true
	}

	for _, member := range artist.Members {
		if strings.Contains(strings.ToLower(member), normalizedQuery) {
			return true
		}
	}

	creationYear := strconv.Itoa(artist.CreationYear)
	if strings.Contains(creationYear, normalizedQuery) {
		return true
	}

	if strings.Contains(strings.ToLower(artist.FirstAlbum), normalizedQuery) {
		return true
	}

	for _, country := range artist.Countries {
		if strings.Contains(strings.ToLower(country), normalizedQuery) {
			return true
		}
	}

	for _, concert := range artist.Concerts {
		if locationMatches(concert.Location, normalizedQuery) {
			return true
		}
	}

	return false
}

func isEmptyFilter(filters ArtistFilterParams) bool {
	return filters.CreationYearFrom == nil &&
		filters.CreationYearTo == nil &&
		filters.FirstAlbumYearFrom == nil &&
		filters.FirstAlbumYearTo == nil &&
		len(filters.MemberCounts) == 0 &&
		len(filters.Countries) == 0
}

func locationMatches(locationName, query string) bool {
	normalizedLocation := normalizeSearchQuery(locationName)
	normalizedQuery := normalizeSearchQuery(query)

	if strings.Contains(normalizedLocation, normalizedQuery) {
		return true
	}

	hyphenatedQuery := strings.ReplaceAll(normalizedQuery, " ", "-")
	if normalizedLocation == hyphenatedQuery {
		return true
	}

	parts := strings.Split(locationName, "-")
	if len(parts) < 2 {
		return false
	}

	country := parts[len(parts)-1]
	city := strings.Join(parts[:len(parts)-1], "-")

	normalizedCity := normalizeSearchQuery(city)
	normalizedCountry := normalizeSearchQuery(country)

	if normalizedQuery == normalizedCity || normalizedQuery == normalizedCountry {
		return true
	}

	cityWithSpaces := strings.ReplaceAll(normalizedCity, "-", " ")
	return normalizedQuery == cityWithSpaces
}

// ============================================================================
// SEARCH CACHE MANAGEMENT
// ============================================================================

func (s *Store) getCachedSearchResults(query string) ([]Artist, bool) {
	s.searchCacheMu.Lock()
	defer s.searchCacheMu.Unlock()

	results, ok := s.searchCache[query]
	if !ok {
		return nil, false
	}

	s.moveKeyToEndLocked(query)
	return results, true
}

func (s *Store) setCachedSearchResults(query string, results []Artist) {
	s.searchCacheMu.Lock()
	defer s.searchCacheMu.Unlock()

	if s.searchCache == nil {
		s.searchCache = make(map[string][]Artist, s.searchCacheSize)
	}
	if s.searchCacheSize <= 0 {
		s.searchCacheSize = 50
	}

	if _, exists := s.searchCache[query]; exists {
		s.searchCache[query] = results
		s.moveKeyToEndLocked(query)
		return
	}

	if len(s.searchOrder) >= s.searchCacheSize {
		oldest := s.searchOrder[0]
		delete(s.searchCache, oldest)
		s.searchOrder = s.searchOrder[1:]
	}

	s.searchCache[query] = results
	s.searchOrder = append(s.searchOrder, query)
}

func (s *Store) moveKeyToEndLocked(query string) {
	for i, key := range s.searchOrder {
		if key == query {
			if i == len(s.searchOrder)-1 {
				return
			}
			copy(s.searchOrder[i:], s.searchOrder[i+1:])
			s.searchOrder[len(s.searchOrder)-1] = query
			return
		}
	}

	s.searchOrder = append(s.searchOrder, query)
}
