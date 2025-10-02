package data

import (
	"context"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"

	"groupie-tracker/internal/api"
)

// Store is the central data repository holding all precomputed application data.
// All collections become immutable after Load() completes, enabling safe concurrent reads without locking.
// This design trades memory for performance - we precompute indexes, suggestions, and metadata at startup
// rather than computing them on every request.
type Store struct {
	apiClient *api.Client // External API client for fetching raw artist and relation data
	withCache bool        // Whether to enable local image caching (set at initialization, never changes)

	cacheEnabled bool // Actual cache status after initialization (may differ from withCache if caching fails)

	// Core data collections - immutable after Load() completes, safe for concurrent reads
	artists         []*Artist             // All artists sorted alphabetically by name
	artistsByID     map[int]*Artist       // O(1) lookup by artist ID
	artistsBySlug   map[string]*Artist    // O(1) lookup by URL-friendly slug (e.g., "pink-floyd")
	artistPositions map[int]int           // Maps artist ID to its index in the sorted artists slice (for navigation)
	locations       []Location            // All concert locations aggregated from artist data
	locationsBySlug map[string]Location   // O(1) lookup by location slug (e.g., "london-uk")
	appStats        AppStats              // Precomputed statistics (total artists, locations, members, concerts, etc.)
	suggestions     []SearchSuggestion    // Precomputed search suggestions for autocomplete (artist names, members, locations)
	artistFilters   ArtistFilterOptions   // Available filter values (creation years, album years, member counts, countries)
	locationFilters LocationFilterOptions // Available location filter values (concert ranges, year ranges, countries)

	// Search result cache (LRU-style) - protects performance on repeated identical queries
	searchCacheMu   sync.Mutex           // Mutex protects concurrent access to cache maps
	searchCache     map[string][]*Artist // Maps normalized query strings to cached result slices
	searchOrder     []string             // Tracks query insertion order for LRU eviction
	searchCacheSize int                  // Maximum cache entries (50) before LRU eviction kicks in

	loadOnce sync.Once // Ensures Load() executes exactly once even if called concurrently
	loadErr  error     // Stores any error from the single Load() execution for return to all callers
}

// NewStore initializes an empty Store with the given API client and caching preference.
// The Store is not usable until Load() successfully completes.
func NewStore(apiClient *api.Client, withCache bool) *Store {
	return &Store{
		apiClient:       apiClient,
		withCache:       withCache,
		searchCache:     make(map[string][]*Artist, 50), // Pre-allocate for 50 entries (LRU cache size)
		searchOrder:     make([]string, 0, 50),          // Pre-allocate for 50 entries (LRU cache size)
		searchCacheSize: 50,                             // Max cached searches before eviction
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

// loadData performs the multi-stage data loading pipeline with concurrent API fetching and processing.
// Stage 1: Fetch artists and relations concurrently using goroutines and channels
// Stage 2: Transform raw API data into rich domain models with computed fields
// Stage 3: Optionally cache images using adaptive worker pool (scales with CPU cores)
// Stage 4: Build indexes, metadata, and search suggestions concurrently for fast startup
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

	// Stage 3: Optional image caching with adaptive worker pool (scales with CPU cores for efficiency)
	var cachedImages, downloadedImages int
	if s.withCache {
		var cacheEnabled bool
		cacheEnabled, cachedImages, downloadedImages = s.cacheImages(artists) // Returns stats for logging
		s.cacheEnabled = cacheEnabled                                         // May be false if caching setup failed
	} else {
		s.cacheEnabled = false
	}

	s.artists = artists // Store the processed artists (sorted alphabetically by name)

	// Stage 4: Build all indexes, metadata, and suggestions concurrently for fast startup
	// These computations are CPU-bound and independent, so parallelizing them reduces total startup time
	var (
		artistsByID     map[int]*Artist
		artistsBySlug   map[string]*Artist
		artistPositions map[int]int
		locations       []Location
		locationsBySlug map[string]Location
		artistFilters   ArtistFilterOptions
		locationFilters LocationFilterOptions
		suggestions     []SearchSuggestion
	)

	var wg sync.WaitGroup // Wait for all concurrent processing to complete

	wg.Add(1)
	go func() { // Goroutine 1: Build artist lookup indexes (ID, slug, position)
		defer wg.Done()
		artistsByID, artistsBySlug, artistPositions = s.createArtistIndexes(artists)
	}()

	wg.Add(1)
	go func() { // Goroutine 2: Aggregate location data from all artists and build location indexes
		defer wg.Done()
		locations, locationsBySlug = s.createLocationsData(artists)
		locationFilters = s.calculateLocationFilterOptions(locations) // Dependent on locations, so done in same goroutine
	}()

	wg.Add(1)
	go func() { // Goroutine 3: Calculate available filter options for artist filtering UI
		defer wg.Done()
		artistFilters = s.calculateArtistFilterOptions(artists)
	}()

	wg.Add(1)
	go func() { // Goroutine 4: Generate search suggestions for autocomplete (artist names, members, locations)
		defer wg.Done()
		suggestions = s.generateSearchSuggestions(artists)
	}()

	wg.Wait() // Block until all goroutines complete

	// Store all computed data - from this point forward, all fields are immutable and thread-safe
	s.artistsByID = artistsByID
	s.artistsBySlug = artistsBySlug
	s.artistPositions = artistPositions
	s.locations = locations
	s.locationsBySlug = locationsBySlug
	s.artistFilters = artistFilters
	s.locationFilters = locationFilters
	s.suggestions = suggestions
	s.appStats = s.calculateStats(artists, locations, cachedImages, downloadedImages) // Final stats computation

	return nil
}

// ============================================================================
// READ-ONLY ACCESSORS - Thread-safe after Load() completes
// ============================================================================

// Artists returns the complete artist collection sorted alphabetically by name.
// Safe for concurrent access after Load() completes since the slice is immutable.
func (s *Store) Artists() []*Artist {
	return s.artists
}

// ArtistByID performs O(1) lookup of an artist by their unique ID.
// Returns the artist and true if found, or nil and false if not found.
func (s *Store) ArtistByID(id int) (*Artist, bool) {
	artist, ok := s.artistsByID[id]
	return artist, ok
}

// ArtistBySlug performs O(1) lookup of an artist by their URL-friendly slug (e.g., "pink-floyd").
// Returns the artist and true if found, or nil and false if not found.
func (s *Store) ArtistBySlug(slug string) (*Artist, bool) {
	artist, ok := s.artistsBySlug[slug]
	return artist, ok
}

// ArtistPosition returns the zero-based index position of an artist within the sorted artists slice.
// Useful for implementing "previous/next artist" navigation in the UI.
// Returns the index and true if found, or -1 and false if the artist ID doesn't exist.
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
// DATA PROCESSING & LOADING
// ============================================================================

// processArtists transforms raw API data into enriched Artist domain models.
func (s *Store) processArtists(apiArtists []api.Artist, apiRelations api.Relation) []*Artist {
	artists := s.transformAPIArtists(apiArtists)
	artists = s.addConcertData(artists, apiRelations)
	return artists
}

// transformAPIArtists converts raw API artist data to domain Artist objects.
func (s *Store) transformAPIArtists(apiArtists []api.Artist) []*Artist {
	artists := make([]*Artist, 0, len(apiArtists))

	for _, apiArtist := range apiArtists {
		artist := &Artist{
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
func (s *Store) addConcertData(artists []*Artist, apiRelations api.Relation) []*Artist {
	// Index relations by artist ID for efficient lookup
	relationMap := make(map[int]api.RelationIndex)
	for _, rel := range apiRelations.Index {
		relationMap[rel.ID] = rel
	}

	// Add concert data to each artist
	for i := range artists {
		artist := artists[i]

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
func (s *Store) createArtistIndexes(artists []*Artist) (map[int]*Artist, map[string]*Artist, map[int]int) {
	artistsByID := make(map[int]*Artist, len(artists))
	artistsBySlug := make(map[string]*Artist, len(artists))
	positions := make(map[int]int, len(artists))

	for idx, artist := range artists {
		artistsByID[artist.ID] = artist
		artistsBySlug[artist.Slug] = artist
		positions[artist.ID] = idx
	}

	return artistsByID, artistsBySlug, positions
}

// createLocationsData builds location aggregates and lookup maps.
func (s *Store) createLocationsData(artists []*Artist) ([]Location, map[string]Location) {
	locations := s.createLocations(artists)
	locationsBySlug := make(map[string]Location, len(locations))
	for _, location := range locations {
		locationsBySlug[location.Slug] = location
	}
	return locations, locationsBySlug
}

// createLocations builds location models from artist concert data.
func (s *Store) createLocations(artists []*Artist) []Location {
	// Build lookup map once - O(n) instead of O(n²)
	artistMap := make(map[int]*Artist, len(artists))
	for _, artist := range artists {
		artistMap[artist.ID] = artist
	}

	locationMap := make(map[string]*Location)
	// Track concert count per artist per location
	artistConcertCount := make(map[string]map[int]int)

	for _, artist := range artists {
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
// STATISTICS CALCULATION
// ============================================================================

// calculateStats computes application statistics from derived data.
func (s *Store) calculateStats(artists []*Artist, locations []Location, cachedImages, downloadedImages int) AppStats {
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
// HELPER FUNCTIONS
// ============================================================================

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
