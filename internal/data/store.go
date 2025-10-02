package data

import (
	"context"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"groupie-tracker/internal/api"
)

// Store is the central data repository holding all precomputed application data.
// All collections become immutable after Load() completes, enabling safe concurrent reads without locking.
// Store now delegates core data access to Catalog and focuses on filters, search, and caching.
type Store struct {
	apiClient *api.Client // External API client for fetching raw artist and relation data

	// Core data - delegated to Catalog
	catalog *Catalog // Owns all normalized data (artists, locations, concerts) and provides query methods

	// Computed metadata and filters - immutable after Load() completes
	appStats        AppStats              // Precomputed statistics (total artists, locations, members, concerts, etc.)
	suggestions     []SearchSuggestion    // Precomputed search suggestions for autocomplete (artist names, members, locations)
	artistFilters   ArtistFilterOptions   // Available filter values (creation years, album years, member counts, countries)
	locationFilters LocationFilterOptions // Available location filter values (concert ranges, year ranges, countries)

	loadOnce sync.Once // Ensures Load() executes exactly once even if called concurrently
	loadErr  error     // Stores any error from the single Load() execution for return to all callers
}

// NewStore initializes an empty Store with the given API client.
// Image caching is always enabled. The Store is not usable until Load() successfully completes.
func NewStore(apiClient *api.Client) *Store {
	return &Store{
		apiClient: apiClient,
	}
}

// Load orchestrates the entire data loading pipeline: fetch API data, process into domain models,
// build indexes, compute metadata, and cache images if enabled. This method is thread-safe and
// executes exactly once via sync.Once, even if called concurrently from multiple goroutines.
func (s *Store) Load(ctx context.Context) error {
	s.loadOnce.Do(func() {
		s.loadErr = s.loadData(ctx) // Actual loading happens in loadData
	})
	return s.loadErr // Return the stored result from the single execution
}

// loadData performs the data loading pipeline in clear stages:
// Stage 1: Fetch raw data from API (artists and relations concurrently)
// Stage 2: Normalize and enrich into domain models
// Stage 3: Optional image caching
// Stage 4: Build Catalog with normalized data
// Stage 5: Compute metadata and filters (concurrently)
func (s *Store) loadData(ctx context.Context) error {
	// Stage 1: Concurrent API fetching - artists and relations fetched in parallel to minimize wait time
	artistsCh := make(chan struct {
		data []api.Artist
		err  error
	}, 1) // Buffered channel prevents goroutine blocking
	relationsCh := make(chan struct {
		data api.Relation
		err  error
	}, 1) // Buffered channel prevents goroutine blocking

	go func() { // Goroutine 1: Fetch artists in parallel with relations fetch
		data, err := s.apiClient.FetchArtists(ctx)
		artistsCh <- struct {
			data []api.Artist
			err  error
		}{data, err}
	}()

	go func() { // Goroutine 2: Fetch relations in parallel with artists fetch
		data, err := s.apiClient.FetchRelations(ctx)
		relationsCh <- struct {
			data api.Relation
			err  error
		}{data, err}
	}()

	// Wait for both fetches to complete - order doesn't matter, both must succeed
	artistsResult := <-artistsCh
	relationsResult := <-relationsCh

	if artistsResult.err != nil {
		return fmt.Errorf("failed to fetch artists: %w", artistsResult.err)
	}
	if relationsResult.err != nil {
		return fmt.Errorf("failed to fetch relations: %w", relationsResult.err)
	}

	// Stage 2: Transform raw API models into rich domain models with computed fields
	artists := s.processArtists(artistsResult.data, relationsResult.data)

	// Stage 3: Image caching with adaptive worker pool (scales with CPU cores for efficiency)
	// Always cache images locally to static/img/artists directory
	var cachedImages, downloadedImages int
	var cacheEnabled bool
	cacheEnabled, cachedImages, downloadedImages = s.cacheImages(artists) // Returns stats for logging
	_ = cacheEnabled                                                      // Cache status tracked for potential future use

	// Stage 4: Build Catalog with normalized data
	catalog := NewCatalog()
	for _, artist := range artists {
		catalog.AddArtist(artist)
		// Concerts will be extracted from artist.Concerts during Build()
	}

	if err := catalog.Build(); err != nil {
		return fmt.Errorf("failed to build catalog: %w", err)
	}

	s.catalog = catalog

	// Stage 5: Compute metadata and filters (can still be done concurrently)
	var (
		artistFilters   ArtistFilterOptions
		locationFilters LocationFilterOptions
		suggestions     []SearchSuggestion
	)

	var wg sync.WaitGroup // Wait for all concurrent processing to complete

	wg.Add(1)
	go func() { // Goroutine 1: Calculate available filter options for artist filtering UI
		defer wg.Done()
		artistFilters = s.calculateArtistFilterOptions(artists)
	}()

	wg.Add(1)
	go func() { // Goroutine 2: Calculate location filter options
		defer wg.Done()
		locationFilters = s.calculateLocationFilterOptions(catalog.AllLocations())
	}()

	wg.Add(1)
	go func() { // Goroutine 3: Generate search suggestions for autocomplete (artist names, members, locations)
		defer wg.Done()
		suggestions = s.generateSearchSuggestions(artists)
	}()

	wg.Wait() // Block until all goroutines complete

	// Store all computed metadata - from this point forward, all fields are immutable and thread-safe
	s.artistFilters = artistFilters
	s.locationFilters = locationFilters
	s.suggestions = suggestions
	s.appStats = s.calculateStats(artists, catalog.AllLocations(), cachedImages, downloadedImages) // Final stats computation

	return nil
}

// ============================================================================
// READ-ONLY ACCESSORS - Thread-safe after Load() completes
// ============================================================================

// Artists returns the complete artist collection sorted alphabetically by name.
// Safe for concurrent access after Load() completes since the slice is immutable.
func (s *Store) Artists() []*Artist {
	if s.catalog == nil {
		return nil
	}
	return s.catalog.AllArtists()
}

// ArtistByID performs O(1) lookup of an artist by their unique ID.
// Returns the artist and true if found, or nil and false if not found.
func (s *Store) ArtistByID(id int) (*Artist, bool) {
	if s.catalog == nil {
		return nil, false
	}
	artist, err := s.catalog.ArtistByID(id)
	return artist, err == nil
}

// ArtistBySlug performs O(1) lookup of an artist by their URL-friendly slug (e.g., "pink-floyd").
// Returns the artist and true if found, or nil and false if not found.
func (s *Store) ArtistBySlug(slug string) (*Artist, bool) {
	if s.catalog == nil {
		return nil, false
	}
	artist, err := s.catalog.ArtistBySlug(slug)
	return artist, err == nil
}

// ArtistPosition returns the zero-based index position of an artist within the sorted artists slice.
// Useful for implementing "previous/next artist" navigation in the UI.
// Returns the index and true if found, or -1 and false if the artist ID doesn't exist.
func (s *Store) ArtistPosition(id int) (int, bool) {
	if s.catalog == nil {
		return -1, false
	}
	pos := s.catalog.ArtistPosition(id)
	return pos, pos >= 0
}

// Locations returns all locations sorted by concert count.
func (s *Store) Locations() []Location {
	if s.catalog == nil {
		return nil
	}
	return s.catalog.AllLocations()
}

// LocationBySlug returns a location by URL slug, or false if not found.
func (s *Store) LocationBySlug(slug string) (Location, bool) {
	if s.catalog == nil {
		return Location{}, false
	}
	location, err := s.catalog.LocationBySlug(slug)
	return location, err == nil
}

// Stats returns application statistics.
func (s *Store) Stats() AppStats {
	return s.appStats
}

// Suggestions returns the precomputed search suggestions for autocomplete.
func (s *Store) Suggestions() []SearchSuggestion {
	return s.suggestions
}

// ArtistFilterOptions returns the precomputed artist filter metadata.
func (s *Store) ArtistFilterOptions() ArtistFilterOptions {
	return s.artistFilters
}

// LocationFilterOptions returns the precomputed location filter metadata.
func (s *Store) LocationFilterOptions() LocationFilterOptions {
	return s.locationFilters
}

// ============================================================================
// DATA PROCESSING & LOADING
// ============================================================================

// processArtists transforms raw API data into enriched Artist domain models.
// This is the main orchestrator for data transformation.
func (s *Store) processArtists(apiArtists []api.Artist, apiRelations api.Relation) []*Artist {
	artists := normalizeAPIArtists(apiArtists)
	artists = enrichWithConcerts(artists, apiRelations)
	return artists
}

// normalizeAPIArtists converts raw API artist data to domain Artist objects.
// This is a pure function with no side effects.
func normalizeAPIArtists(apiArtists []api.Artist) []*Artist {
	artists := make([]*Artist, 0, len(apiArtists))

	for _, apiArtist := range apiArtists {
		artist := &Artist{
			ID:           apiArtist.ID,
			Name:         apiArtist.Name,
			Members:      apiArtist.Members,
			CreationYear: apiArtist.CreationYear,
			FirstAlbum:   apiArtist.FirstAlbum,
			Image:        apiArtist.Image,
		}
		artists = append(artists, artist)
	}

	return artists
}

// enrichWithConcerts adds concert information from API relations to artists.
// Concerts are parsed, normalized, and sorted chronologically.
func enrichWithConcerts(artists []*Artist, apiRelations api.Relation) []*Artist {
	// Index relations by artist ID for efficient lookup
	relationMap := make(map[int]api.RelationIndex, len(apiRelations.Index))
	for _, rel := range apiRelations.Index {
		relationMap[rel.ID] = rel
	}

	// Add concert data to each artist
	for i := range artists {
		artist := artists[i]

		if rel, exists := relationMap[artist.ID]; exists {
			artist.Concerts = normalizeConcerts(artist.ID, rel.DatesLocations)
		}
	}

	// Sort artists by name for consistent display
	sort.Slice(artists, func(i, j int) bool {
		return artists[i].Name < artists[j].Name
	})

	return artists
}

// normalizeConcerts converts raw location-date mappings into Concert objects.
// Dates are parsed into time.Time, locations are normalized, and concerts are sorted chronologically.
func normalizeConcerts(artistID int, datesLocations map[string][]string) []Concert {
	concerts := make([]Concert, 0)

	for location, dates := range datesLocations {
		normalizedLoc := normalizeLocation(location)
		locationSlug := createSlug(normalizedLoc)

		for _, dateStr := range dates {
			parsedDate, err := parseDate(dateStr)
			if err != nil {
				// If parsing fails, use zero date but keep the original string
				parsedDate = time.Time{}
			}

			concerts = append(concerts, Concert{
				ArtistID:     artistID,
				Location:     normalizedLoc,
				LocationSlug: locationSlug,
				Date:         parsedDate,
				DateString:   dateStr,
			})
		}
	}

	// Sort concerts chronologically
	sort.Slice(concerts, func(i, j int) bool {
		return concerts[i].Date.Before(concerts[j].Date)
	})

	return concerts
}

// createArtistIndexes builds lookup maps for artists by ID and slug.
// ============================================================================
// STATISTICS CALCULATION
// ============================================================================

// calculateStats computes application statistics from derived data.
func (s *Store) calculateStats(artists []*Artist, locations []Location, cachedImages, downloadedImages int) AppStats {
	totalMembers := 0
	totalConcerts := 0
	countries := make(map[string]bool)

	for _, artist := range artists {
		totalMembers += len(artist.Members)
		totalConcerts += artist.ConcertCount()

		for _, country := range artist.Countries() {
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
// HELPER FUNCTIONS
// ============================================================================

var (
	slugRegex     *regexp.Regexp
	slugRegexOnce sync.Once
)

// getSlugRegex returns the compiled regex for slug creation, initializing it once.
func getSlugRegex() *regexp.Regexp {
	slugRegexOnce.Do(func() {
		slugRegex = regexp.MustCompile(`[^a-z0-9]+`)
	})
	return slugRegex
}

// createSlug converts display names into URL-friendly slugs.
func createSlug(name string) string {
	reg := getSlugRegex()
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

// parseDate parses a date string in DD-MM-YYYY format and returns a time.Time.
// Returns zero time and an error if parsing fails.
func parseDate(dateStr string) (time.Time, error) {
	dateStr = strings.TrimSpace(dateStr)
	if len(dateStr) == 0 {
		return time.Time{}, fmt.Errorf("empty date string")
	}

	// Try DD-MM-YYYY format (most common in the API)
	if t, err := time.Parse("02-01-2006", dateStr); err == nil {
		return t, nil
	}

	// Try YYYY-MM-DD format
	if t, err := time.Parse("2006-01-02", dateStr); err == nil {
		return t, nil
	}

	// Try DD/MM/YYYY format
	if t, err := time.Parse("02/01/2006", dateStr); err == nil {
		return t, nil
	}

	return time.Time{}, fmt.Errorf("unable to parse date: %s", dateStr)
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
