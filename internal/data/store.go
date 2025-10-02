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

// ============================================================================
// READ-ONLY ACCESSORS
// ============================================================================

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
