package data

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

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

// ============================================================================
// DATA PROCESSING & LOADING
// ============================================================================

// processArtists transforms raw API data into enriched Artist domain models.
func (s *Store) processArtists(apiArtists []api.Artist, apiRelations api.Relation) []Artist {
	artists := s.transformAPIArtists(apiArtists)
	artists = s.addConcertData(artists, apiRelations)
	return artists
}

// transformAPIArtists converts raw API artist data to domain Artist objects.
func (s *Store) transformAPIArtists(apiArtists []api.Artist) []Artist {
	artists := make([]Artist, 0, len(apiArtists))

	for _, apiArtist := range apiArtists {
		artist := Artist{
			ID:              apiArtist.ID,
			Name:            apiArtist.Name,
			Slug:            createSlug(apiArtist.Name),
			Members:         apiArtist.Members,
			CreationYear:    apiArtist.CreationYear,
			FirstAlbum:      apiArtist.FirstAlbum,
			Image:           apiArtist.Image,
			DatesAtLocation: make(map[string][]string),
			MemberCount:     len(apiArtist.Members),
			FirstAlbumYear:  extractYearFromDate(apiArtist.FirstAlbum),
		}
		artists = append(artists, artist)
	}

	return artists
}

// addConcertData enriches artists with concert information from API relations.
func (s *Store) addConcertData(artists []Artist, apiRelations api.Relation) []Artist {
	// Index relations by artist ID for efficient lookup
	relationMap := make(map[int]api.RelationIndex)
	for _, rel := range apiRelations.Index {
		relationMap[rel.ID] = rel
	}

	// Add concert data to each artist
	for i := range artists {
		artist := &artists[i]

		if rel, exists := relationMap[artist.ID]; exists {
			countries := make(map[string]bool)

			// Process each location and its dates
			for location, dates := range rel.DatesLocations {
				normalizedLoc := normalizeLocation(location)
				locationSlug := createSlug(normalizedLoc)
				artist.DatesAtLocation[locationSlug] = dates

				// Create concert objects
				for _, date := range dates {
					artist.Concerts = append(artist.Concerts, Concert{
						Date:     date,
						Location: normalizedLoc,
					})
				}

				// Extract country from location
				countries[extractCountryFromLocation(normalizedLoc)] = true
			}

			// Sort concerts chronologically
			sort.Slice(artist.Concerts, func(i, j int) bool {
				return artist.Concerts[i].Date < artist.Concerts[j].Date
			})

			// Set derived fields
			artist.ConcertCount = len(artist.Concerts)
			artist.Countries = s.convertCountriesMapToSlice(countries)
		}
	}

	// Sort artists by name for consistent display
	sort.Slice(artists, func(i, j int) bool {
		return artists[i].Name < artists[j].Name
	})

	return artists
}

// createArtistIndexes builds lookup maps for artists by ID and slug.
func (s *Store) createArtistIndexes(artists []Artist) (map[int]Artist, map[string]Artist, map[int]int) {
	artistsByID := make(map[int]Artist, len(artists))
	artistsBySlug := make(map[string]Artist, len(artists))
	positions := make(map[int]int, len(artists))

	for idx, artist := range artists {
		artistsByID[artist.ID] = artist
		artistsBySlug[artist.Slug] = artist
		positions[artist.ID] = idx
	}

	return artistsByID, artistsBySlug, positions
}

// createLocationsData builds location aggregates and lookup maps.
func (s *Store) createLocationsData(artists []Artist) ([]Location, map[string]Location) {
	locations := s.createLocations(artists)
	locationsBySlug := make(map[string]Location, len(locations))
	for _, location := range locations {
		locationsBySlug[location.Slug] = location
	}
	return locations, locationsBySlug
}

// createLocations builds location models from artist concert data.
func (s *Store) createLocations(artists []Artist) []Location {
	// Build lookup map once - O(n) instead of O(n²)
	artistMap := make(map[int]Artist, len(artists))
	for _, artist := range artists {
		artistMap[artist.ID] = artist
	}

	locationMap := make(map[string]*Location)
	// Track concert count per artist per location
	artistConcertCount := make(map[string]map[int]int)

	for i := range artists {
		artist := &artists[i]
		for _, concert := range artist.Concerts {
			// Initialize location if not exists
			if _, exists := locationMap[concert.Location]; !exists {
				locationMap[concert.Location] = &Location{
					Name:         concert.Location,
					Slug:         createSlug(concert.Location),
					Country:      extractCountryFromLocation(concert.Location),
					Artists:      make([]ArtistAtLocation, 0),
					EarliestYear: 9999, // Initialize with high value
					LatestYear:   0,    // Initialize with low value
				}
				artistConcertCount[concert.Location] = make(map[int]int)
			}

			// Count concerts per artist per location
			artistConcertCount[concert.Location][artist.ID]++
			locationMap[concert.Location].TotalConcerts++

			// Update year range for this location
			year := extractYearFromDate(concert.Date)
			if year > 0 {
				if year < locationMap[concert.Location].EarliestYear {
					locationMap[concert.Location].EarliestYear = year
				}
				if year > locationMap[concert.Location].LatestYear {
					locationMap[concert.Location].LatestYear = year
				}
			}
		}
	}

	// Convert concert count map to ArtistAtLocation structs
	for locationName, location := range locationMap {
		artistCounts := artistConcertCount[locationName]
		artistsAtLocation := make([]ArtistAtLocation, 0, len(artistCounts))

		for artistID, concertCount := range artistCounts {
			// Use O(1) map lookup instead of O(n) linear search
			if artist, found := artistMap[artistID]; found {
				artistsAtLocation = append(artistsAtLocation, ArtistAtLocation{
					Artist:       artist,
					ConcertCount: concertCount,
				})
			}
		}

		// Sort artists by concert count (descending), then by name
		sort.Slice(artistsAtLocation, func(i, j int) bool {
			if artistsAtLocation[i].ConcertCount != artistsAtLocation[j].ConcertCount {
				return artistsAtLocation[i].ConcertCount > artistsAtLocation[j].ConcertCount
			}
			return artistsAtLocation[i].Artist.Name < artistsAtLocation[j].Artist.Name
		})

		location.Artists = artistsAtLocation
		location.ArtistCount = len(artistsAtLocation)
	}

	// Convert to slice and sort by concert count
	locations := make([]Location, 0, len(locationMap))
	for _, loc := range locationMap {
		locations = append(locations, *loc)
	}

	sort.Slice(locations, func(i, j int) bool {
		return locations[i].TotalConcerts > locations[j].TotalConcerts
	})

	return locations
}

// ============================================================================
// FILTER OPTIONS CALCULATION
// ============================================================================

// calculateArtistFilterOptions derives available artist filter metadata from the dataset.
func (s *Store) calculateArtistFilterOptions(artists []Artist) ArtistFilterOptions {
	if len(artists) == 0 {
		return ArtistFilterOptions{}
	}

	minCreationYear, maxCreationYear := artists[0].CreationYear, artists[0].CreationYear
	minFirstAlbumYear, maxFirstAlbumYear := 0, 0
	memberCountSet := make(map[int]bool)
	countrySet := make(map[string]bool)

	for _, artist := range artists {
		if artist.CreationYear < minCreationYear {
			minCreationYear = artist.CreationYear
		}
		if artist.CreationYear > maxCreationYear {
			maxCreationYear = artist.CreationYear
		}

		albumYear := artist.FirstAlbumYear
		if albumYear == 0 {
			albumYear = extractYearFromDate(artist.FirstAlbum)
		}
		if albumYear > 0 {
			if minFirstAlbumYear == 0 || albumYear < minFirstAlbumYear {
				minFirstAlbumYear = albumYear
			}
			if albumYear > maxFirstAlbumYear {
				maxFirstAlbumYear = albumYear
			}
		}

		memberCount := artist.MemberCount
		if memberCount == 0 {
			memberCount = len(artist.Members)
		}
		memberCountSet[memberCount] = true

		for _, country := range artist.Countries {
			if country != "" {
				countrySet[country] = true
			}
		}
	}

	memberCounts := make([]int, 0, len(memberCountSet))
	for count := range memberCountSet {
		memberCounts = append(memberCounts, count)
	}
	sort.Ints(memberCounts)

	countries := make([]string, 0, len(countrySet))
	for country := range countrySet {
		countries = append(countries, country)
	}
	sort.Strings(countries)

	if minFirstAlbumYear == 0 {
		minFirstAlbumYear = minCreationYear
	}
	if maxFirstAlbumYear == 0 {
		maxFirstAlbumYear = maxCreationYear
	}

	return ArtistFilterOptions{
		CreationYearMin:   minCreationYear,
		CreationYearMax:   maxCreationYear,
		FirstAlbumYearMin: minFirstAlbumYear,
		FirstAlbumYearMax: maxFirstAlbumYear,
		MemberCounts:      memberCounts,
		Countries:         countries,
	}
}

// calculateLocationFilterOptions derives available location filter metadata.
func (s *Store) calculateLocationFilterOptions(locations []Location) LocationFilterOptions {
	if len(locations) == 0 {
		return LocationFilterOptions{}
	}

	minConcerts, maxConcerts := locations[0].TotalConcerts, locations[0].TotalConcerts
	minArtists, maxArtists := locations[0].ArtistCount, locations[0].ArtistCount
	minYear, maxYear := locations[0].EarliestYear, locations[0].LatestYear
	countrySet := make(map[string]bool)

	for _, location := range locations {
		if location.TotalConcerts < minConcerts {
			minConcerts = location.TotalConcerts
		}
		if location.TotalConcerts > maxConcerts {
			maxConcerts = location.TotalConcerts
		}

		if location.ArtistCount < minArtists {
			minArtists = location.ArtistCount
		}
		if location.ArtistCount > maxArtists {
			maxArtists = location.ArtistCount
		}

		if location.EarliestYear > 0 && location.EarliestYear < minYear {
			minYear = location.EarliestYear
		}
		if location.LatestYear > maxYear {
			maxYear = location.LatestYear
		}

		country := location.Country
		if country == "" {
			country = extractCountryFromLocation(location.Name)
		}
		if country != "" {
			countrySet[country] = true
		}
	}

	countries := make([]string, 0, len(countrySet))
	for country := range countrySet {
		countries = append(countries, country)
	}
	sort.Strings(countries)

	return LocationFilterOptions{
		ConcertCountMin: minConcerts,
		ConcertCountMax: maxConcerts,
		ArtistCountMin:  minArtists,
		ArtistCountMax:  maxArtists,
		ConcertYearMin:  minYear,
		ConcertYearMax:  maxYear,
		Countries:       countries,
	}
}

// ============================================================================
// SEARCH SUGGESTIONS GENERATION
// ============================================================================

// generateSearchSuggestions pre-computes autocomplete suggestions from the dataset.
func newSearchSuggestion(text, suggestionType, description, url string, artistID int) SearchSuggestion {
	return SearchSuggestion{
		Text:           text,
		Type:           SearchSuggestionType(suggestionType),
		Description:    description,
		URL:            url,
		ArtistID:       artistID,
		normalizedText: strings.ToLower(text),
	}
}

func (s *Store) generateSearchSuggestions(artists []Artist) []SearchSuggestion {
	var suggestions []SearchSuggestion
	seen := make(map[string]bool)

	for _, artist := range artists {
		artistKey := "artist:" + artist.Name
		if !seen[artistKey] {
			suggestions = append(suggestions, newSearchSuggestion(
				artist.Name+" - artist",
				string(SuggestionTypeArtist),
				artist.Name+" - artist",
				"/artists/"+artist.Slug,
				artist.ID,
			))
			seen[artistKey] = true
		}

		for _, member := range artist.Members {
			memberKey := "member:" + member
			if !seen[memberKey] {
				suggestions = append(suggestions, newSearchSuggestion(
					member+" - member",
					string(SuggestionTypeMember),
					member+" - member of "+artist.Name,
					"/artists/"+artist.Slug,
					artist.ID,
				))
				seen[memberKey] = true
			}
		}

		for location := range artist.DatesAtLocation {
			locationKey := "location:" + location
			if !seen[locationKey] {
				suggestions = append(suggestions, newSearchSuggestion(
					location+" - location",
					string(SuggestionTypeLocation),
					location+" - concert location",
					"/search?q="+location,
					0,
				))
				seen[locationKey] = true
			}
		}

		creationYear := strconv.Itoa(artist.CreationYear)
		yearKey := "creation:" + creationYear
		if !seen[yearKey] {
			suggestions = append(suggestions, newSearchSuggestion(
				creationYear+" - creation year",
				string(SuggestionTypeCreation),
				"Artists created in "+creationYear,
				"/search?q="+creationYear,
				0,
			))
			seen[yearKey] = true
		}

		albumKey := "album:" + artist.FirstAlbum
		if !seen[albumKey] {
			suggestions = append(suggestions, newSearchSuggestion(
				artist.FirstAlbum+" - first album",
				string(SuggestionTypeFirstAlbum),
				"Albums released on "+artist.FirstAlbum,
				"/search?q="+artist.FirstAlbum,
				0,
			))
			seen[albumKey] = true
		}
	}

	sort.Slice(suggestions, func(i, j int) bool {
		if suggestions[i].Type != suggestions[j].Type {
			return suggestions[i].Type < suggestions[j].Type
		}
		return suggestions[i].Text < suggestions[j].Text
	})

	return suggestions
}

// ============================================================================
// STATISTICS CALCULATION
// ============================================================================

// calculateStats computes application statistics from derived data.
func (s *Store) calculateStats(artists []Artist, locations []Location, cachedImages, downloadedImages int) AppStats {
	totalMembers := 0
	totalConcerts := 0
	countries := make(map[string]bool)

	for _, artist := range artists {
		totalMembers += len(artist.Members)
		totalConcerts += artist.ConcertCount

		for _, country := range artist.Countries {
			countries[country] = true
		}
	}

	return AppStats{
		TotalArtists:     len(artists),
		TotalMembers:     totalMembers,
		TotalLocations:   len(locations),
		TotalConcerts:    totalConcerts,
		TotalCountries:   len(countries),
		CachedImages:     cachedImages,
		DownloadedImages: downloadedImages,
	}
}

// ============================================================================
// IMAGE CACHING
// ============================================================================

// cacheImages downloads and caches artist images locally when caching is enabled.
// Returns whether caching was enabled along with cached/downloaded image counts.
func (s *Store) cacheImages(artists []Artist) (bool, int, int) {
	if !s.withCache {
		return false, 0, 0
	}

	cacheDir := "static/img/artists"
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return false, 0, 0
	}

	// Adaptive worker count: scale with CPU cores, but cap at artist count
	numWorkers := runtime.NumCPU()
	if numWorkers > len(artists) {
		numWorkers = len(artists)
	}
	if numWorkers < 1 {
		numWorkers = 1
	}

	// Job represents a download task
	type job struct {
		index     int
		artist    *Artist
		fileName  string
		filePath  string
		localPath string
		exists    bool
	}

	// Create job queue
	jobs := make(chan job, len(artists))

	// Prepare all jobs
	for i := range artists {
		artist := &artists[i]
		fileName := fmt.Sprintf("%s.jpg", artist.Slug)
		filePath := filepath.Join(cacheDir, fileName)
		localPath := "/" + filepath.ToSlash(filePath)
		exists := false

		// Check if file already exists
		if _, err := os.Stat(filePath); err == nil {
			exists = true
		}

		jobs <- job{
			index:     i,
			artist:    artist,
			fileName:  fileName,
			filePath:  filePath,
			localPath: localPath,
			exists:    exists,
		}
	}
	close(jobs)

	// Atomic counters for thread-safe counting
	var cached, downloaded int32
	var mu sync.Mutex // Mutex for updating artist images

	// Start adaptive worker pool
	var wg sync.WaitGroup
	for w := 0; w < numWorkers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := range jobs {
				if j.exists {
					// Use cached file
					mu.Lock()
					j.artist.Image = j.localPath
					mu.Unlock()
					atomic.AddInt32(&cached, 1)
				} else {
					// Download image with timeout
					if downloadImage(j.artist.Image, j.filePath) {
						mu.Lock()
						j.artist.Image = j.localPath
						mu.Unlock()
						atomic.AddInt32(&downloaded, 1)
					}
				}
			}
		}()
	}

	wg.Wait()

	return true, int(atomic.LoadInt32(&cached)), int(atomic.LoadInt32(&downloaded))
}

// ============================================================================
// HELPER FUNCTIONS
// ============================================================================

// downloadImage downloads and saves a single image from a URL to local filesystem.
// Uses a 10-second timeout to prevent hanging on slow/dead URLs.
func downloadImage(url, path string) bool {
	if strings.TrimSpace(url) == "" {
		return false
	}

	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	resp, err := client.Get(url)
	if err != nil || resp.StatusCode != http.StatusOK {
		if resp != nil {
			resp.Body.Close()
		}
		return false
	}
	defer resp.Body.Close()

	file, err := os.Create(path)
	if err != nil {
		return false
	}
	defer file.Close()

	_, err = io.Copy(file, resp.Body)
	return err == nil
}

// createSlug converts display names into URL-friendly slugs.
func createSlug(name string) string {
	reg := regexp.MustCompile(`[^a-z0-9]+`)
	slug := reg.ReplaceAllString(strings.ToLower(name), "-")
	return strings.Trim(slug, "-")
}

// normalizeLocation converts raw API location strings to consistent format.
func normalizeLocation(location string) string {
	return strings.ToLower(strings.ReplaceAll(strings.TrimSpace(location), "_", "-"))
}

// convertCountriesMapToSlice converts a map[string]bool to sorted string slice.
func (s *Store) convertCountriesMapToSlice(countriesMap map[string]bool) []string {
	countries := make([]string, 0, len(countriesMap))
	for country := range countriesMap {
		if country != "" { // Skip empty countries
			countries = append(countries, country)
		}
	}
	sort.Strings(countries)
	return countries
}

// extractCountryFromLocation normalizes a location string and returns a display-ready country name.
func extractCountryFromLocation(location string) string {
	parts := strings.Split(strings.ToLower(strings.TrimSpace(location)), "-")
	if len(parts) == 0 {
		return ""
	}

	country := strings.TrimSpace(parts[len(parts)-1])
	if country == "" {
		return ""
	}

	switch country {
	case "usa", "us":
		return "USA"
	case "uk":
		return "UK"
	case "uae":
		return "UAE"
	}

	words := strings.Fields(strings.ReplaceAll(country, "-", " "))
	for i, word := range words {
		if len(word) == 0 {
			continue
		}
		words[i] = strings.ToUpper(word[:1]) + strings.ToLower(word[1:])
	}
	return strings.Join(words, " ")
}

// extractYearFromDate attempts to parse common date formats and return the year component.
func extractYearFromDate(dateStr string) int {
	dateStr = strings.TrimSpace(dateStr)
	if len(dateStr) < 4 {
		return 0
	}

	if len(dateStr) >= 10 && dateStr[2] == '-' && dateStr[5] == '-' {
		if year, err := strconv.Atoi(dateStr[6:10]); err == nil {
			return year
		}
	}

	if year, err := strconv.Atoi(dateStr[:4]); err == nil && year > 1900 && year < 3000 {
		return year
	}

	return 0
}
