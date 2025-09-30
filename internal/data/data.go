package data

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"groupie-tracker/internal/config"
)

// --- External API Data Structures ---
//
// These models represent the exact JSON structure returned by the Groupie Tracker API.
// They are used only for initial data loading and are converted to rich domain models
// immediately after parsing. Field names and types match the API specification exactly.

// APIArtist represents the raw artist data structure from the /api/artists endpoint.
type APIArtist struct {
	ID           int      `json:"id"`           // Unique artist identifier from API
	Name         string   `json:"name"`         // Artist/band name as provided by API
	Members      []string `json:"members"`      // Current band member names
	CreationYear int      `json:"creationDate"` // Band formation year (note JSON name mapping)
	FirstAlbum   string   `json:"firstAlbum"`   // First album release date string
	Image        string   `json:"image"`        // Artist image URL from API
}

// APIRelationIndex represents a single artist's concert data from the /api/relation endpoint.
type APIRelationIndex struct {
	ID             int                 `json:"id"`             // Artist ID matching APIArtist.ID
	DatesLocations map[string][]string `json:"datesLocations"` // Raw concert location->dates mapping
}

// APIRelation wraps the complete concert relations dataset from the /api/relation endpoint.
type APIRelation struct {
	Index []APIRelationIndex `json:"index"` // Array of all artist concert relations
}

// --- Core Domain Models ---

// Artist represents the complete internal model of a music artist/band.
type Artist struct {
	ID              int                 // Unique identifier matching API data
	Name            string              // Artist/band name for display
	Slug            string              // URL-friendly identifier (e.g., "queen", "led-zeppelin")
	Members         []string            // Current band member names
	CreationYear    int                 // Band formation year
	FirstAlbum      string              // First album date string (various formats)
	Image           string              // Artist image URL for display
	Concerts        []Concert           // Structured concert events (processed from API relations)
	DatesAtLocation map[string][]string // Pre-indexed concert dates by location slug for fast lookups
	ConcertCount    int                 // Total number of concerts (computed field)
	Countries       []string            // Unique countries where artist performed (sorted, for filtering)
}

// ArtistAtLocation represents an artist's concert activity at a specific venue.
type ArtistAtLocation struct {
	Artist       Artist // Full artist information for display
	ConcertCount int    // Number of concerts this artist held at the location
}

// Location represents the complete internal model of a concert venue.
type Location struct {
	Name          string             // Human-readable location name (e.g., "London UK")
	Slug          string             // URL-friendly identifier (e.g., "london-uk")
	Artists       []ArtistAtLocation // Artists who performed here with concert counts
	ArtistCount   int                // Number of unique artists (computed field)
	TotalConcerts int                // Total concerts held here (computed field)
	EarliestYear  int                // Year of first concert at this location
	LatestYear    int                // Year of most recent concert at this location
}

// Concert represents a single concert event in structured form.
type Concert struct {
	Date     string // Concert date in original API format
	Location string // Normalized location name matching Location.Name
}

// --- Filter Data Structures ---

// ArtistFilterParams represents all possible filter criteria that can be applied to artist searches.
type ArtistFilterParams struct {
	CreationYearFrom   *int     `json:"creationYearFrom,omitempty"`   // Minimum band formation year (inclusive)
	CreationYearTo     *int     `json:"creationYearTo,omitempty"`     // Maximum band formation year (inclusive)
	FirstAlbumYearFrom *int     `json:"firstAlbumYearFrom,omitempty"` // Minimum first album year (inclusive)
	FirstAlbumYearTo   *int     `json:"firstAlbumYearTo,omitempty"`   // Maximum first album year (inclusive)
	MemberCounts       []int    `json:"memberCounts,omitempty"`       // Allowed band member counts (exact match)
	Countries          []string `json:"countries,omitempty"`          // Countries where artist must have performed
}

// ArtistFilterOptions represents the complete set of available filter options for artists.
type ArtistFilterOptions struct {
	CreationYearMin   int      `json:"creationYearMin"`   // Earliest band formation year in dataset
	CreationYearMax   int      `json:"creationYearMax"`   // Latest band formation year in dataset
	FirstAlbumYearMin int      `json:"firstAlbumYearMin"` // Earliest first album year in dataset
	FirstAlbumYearMax int      `json:"firstAlbumYearMax"` // Latest first album year in dataset
	MemberCounts      []int    `json:"memberCounts"`      // All member counts found in dataset (sorted)
	Countries         []string `json:"countries"`         // All countries from concert locations (sorted)
}

// LocationFilterParams represents all possible filter criteria for location searches.
type LocationFilterParams struct {
	ConcertCountFrom *int     `json:"concertCountFrom,omitempty"` // Minimum total concerts held (inclusive)
	ConcertCountTo   *int     `json:"concertCountTo,omitempty"`   // Maximum total concerts held (inclusive)
	ArtistCountFrom  *int     `json:"artistCountFrom,omitempty"`  // Minimum unique artists performed (inclusive)
	ArtistCountTo    *int     `json:"artistCountTo,omitempty"`    // Maximum unique artists performed (inclusive)
	ConcertYearFrom  *int     `json:"concertYearFrom,omitempty"`  // Earliest concert year (inclusive)
	ConcertYearTo    *int     `json:"concertYearTo,omitempty"`    // Latest concert year (inclusive)
	Countries        []string `json:"countries,omitempty"`        // Countries where location must be situated
}

// LocationFilterOptions represents the complete set of available filter options for locations.
type LocationFilterOptions struct {
	ConcertCountMin int      `json:"concertCountMin"` // Minimum concert count across all locations
	ConcertCountMax int      `json:"concertCountMax"` // Maximum concert count across all locations
	ArtistCountMin  int      `json:"artistCountMin"`  // Minimum artist count across all locations
	ArtistCountMax  int      `json:"artistCountMax"`  // Maximum artist count across all locations
	ConcertYearMin  int      `json:"concertYearMin"`  // Earliest concert year across all locations
	ConcertYearMax  int      `json:"concertYearMax"`  // Latest concert year across all locations
	Countries       []string `json:"countries"`       // All countries from location names (sorted)
}

// --- Search Data Structures ---

// SearchSuggestionType represents the type of search result for categorization.
type SearchSuggestionType string

const (
	SuggestionTypeArtist     SearchSuggestionType = "artist"      // Artist/band name
	SuggestionTypeMember     SearchSuggestionType = "member"      // Band member name
	SuggestionTypeLocation   SearchSuggestionType = "location"    // Concert location
	SuggestionTypeFirstAlbum SearchSuggestionType = "first-album" // First album date
	SuggestionTypeCreation   SearchSuggestionType = "creation"    // Band creation date
)

// SearchSuggestion represents a single search suggestion with type identification.
type SearchSuggestion struct {
	Text           string               `json:"text"`        // Display text for the suggestion
	Type           SearchSuggestionType `json:"type"`        // Type of match for categorization
	Description    string               `json:"description"` // Additional context (e.g., "Queen - artist")
	URL            string               `json:"url"`         // Direct link to detail page
	ArtistID       int                  `json:"artistId"`    // Related artist ID for context
	normalizedText string               `json:"-"`           // Lowercase version for efficient matching
}

// SearchResult represents a comprehensive search result with matched items.
type SearchResult struct {
	Artists      []*Artist `json:"artists"`      // Artists matching search criteria (pointers for consistency)
	Query        string    `json:"query"`        // Original search query
	TotalResults int       `json:"totalResults"` // Number of matching results
}

// SearchParams represents search query parameters from user input.
type SearchParams struct {
	Query   string             `json:"query"`   // Search text input
	Filters ArtistFilterParams `json:"filters"` // Optional additional filters
}

// AppStats represents application-wide statistics with type-safe fields.
type AppStats struct {
	TotalArtists     int `json:"total_artists"`     // Number of artists in the dataset
	TotalMembers     int `json:"total_members"`     // Sum of all band members across all artists
	TotalLocations   int `json:"total_locations"`   // Number of unique concert venues
	TotalConcerts    int `json:"total_concerts"`    // Total number of concert events
	TotalCountries   int `json:"total_countries"`   // Number of unique countries with concerts
	CachedImages     int `json:"cached_images"`     // Number of artist images served from local cache
	DownloadedImages int `json:"downloaded_images"` // Number of artist images downloaded this session
}

// ToMap converts the type-safe stats structure to the legacy map format.
func (s AppStats) ToMap() map[string]int {
	return map[string]int{
		"total_artists":     s.TotalArtists,
		"total_members":     s.TotalMembers,
		"total_locations":   s.TotalLocations,
		"total_concerts":    s.TotalConcerts,
		"total_countries":   s.TotalCountries,
		"cached_images":     s.CachedImages,
		"downloaded_images": s.DownloadedImages,
	}
}

// --- DataStore Interface ---

// DataStore represents the simplified data management interface.
// This acts as an adapter to provide a clean interface over Repository.
type DataStore struct {
	Artists             []*Artist           // All artists (sorted alphabetically)
	Locations           []*Location         // All locations
	Stats               Stats               // Data statistics
	ArtistFilterOptions ArtistFilterOptions // Available filter options
	SearchSuggestions   []SearchSuggestion  // Pre-computed search suggestions
	repo                *Repository         // Underlying repository
}

// Stats holds statistical information about the data.
type Stats struct {
	TotalArtists   int
	TotalLocations int
	TotalConcerts  int
	DateRange      DateRange
}

// DateRange represents a range of dates.
type DateRange struct {
	Earliest string
	Latest   string
}

// LoadData loads all data from the API and returns a configured DataStore.
func LoadData(ctx context.Context) (*DataStore, error) {
	repo := NewRepository()
	if err := repo.LoadData(ctx); err != nil {
		return nil, fmt.Errorf("failed to load repository data: %w", err)
	}

	// Create DataStore adapter
	artists := repo.GetArtists()
	locations := repo.GetLocations()

	// Compute statistics
	stats := computeDataStats(artists)

	return &DataStore{
		Artists:             artists,
		Locations:           locations,
		Stats:               stats,
		ArtistFilterOptions: repo.GetArtistFilterOptions(),
		SearchSuggestions:   repo.GenerateAllSearchSuggestions(),
		repo:                repo,
	}, nil
}

// GetArtistByID returns the artist with the given ID and true if found.
func (s *DataStore) GetArtistByID(id int) (*Artist, bool) {
	return s.repo.GetArtistByID(id)
}

// GetArtistBySlug returns the artist with the given slug and true if found.
func (s *DataStore) GetArtistBySlug(slug string) (*Artist, bool) {
	return s.repo.GetArtistBySlug(slug)
}

// GetAdjacentArtists returns the previous and next artists in alphabetical order.
func (s *DataStore) GetAdjacentArtists(artistID int) (*Artist, *Artist) {
	return s.repo.GetAdjacentArtists(artistID)
}

// FilterArtists returns artists matching the given filter parameters.
func (s *DataStore) FilterArtists(params ArtistFilterParams) []*Artist {
	return s.repo.FilterArtists(params)
}

// SearchArtists performs search across artists with optional filtering.
func (s *DataStore) SearchArtists(query string, filters ArtistFilterParams) []*Artist {
	searchParams := SearchParams{
		Query:   query,
		Filters: filters,
	}
	result := s.repo.SearchArtists(searchParams)
	return result.Artists
}

// computeDataStats calculates statistical information about the loaded data.
func computeDataStats(artists []*Artist) Stats {
	stats := Stats{
		TotalArtists: len(artists),
	}

	// Count total concerts and collect dates
	totalConcerts := 0
	var dates []string

	for _, artist := range artists {
		for _, datelist := range artist.DatesAtLocation {
			totalConcerts += len(datelist)
			dates = append(dates, datelist...)
		}
	}

	stats.TotalConcerts = totalConcerts

	// Find date range
	if len(dates) > 0 {
		sort.Strings(dates)
		stats.DateRange = DateRange{
			Earliest: dates[0],
			Latest:   dates[len(dates)-1],
		}
	}

	return stats
}

// --- Repository Implementation ---

// Repository provides centralized data management for the Groupie Tracker application.
type Repository struct {
	artists       []*Artist      // Pointer-based storage for efficient memory usage
	artistIndex   map[int]int    // Artist ID to index mapping for O(1) lookups
	artistSlugMap map[string]int // Artist slug to index mapping for O(1) slug lookups
	locations     []*Location    // All locations with computed statistics
	locationIndex map[string]int // Location slug to index mapping for O(1) lookups
	mu            sync.RWMutex   // Protects data during concurrent access
}

// NewRepository creates a new Repository instance with initialized collections.
func NewRepository() *Repository {
	return &Repository{
		artists:       make([]*Artist, 0),
		artistIndex:   make(map[int]int),
		artistSlugMap: make(map[string]int),
		locations:     make([]*Location, 0),
		locationIndex: make(map[string]int),
	}
}

// LoadData fetches and processes all data from the Groupie Tracker API.
func (r *Repository) LoadData(ctx context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Load raw API data
	apiArtists, apiRelations, err := r.fetchAPIData(ctx)
	if err != nil {
		return fmt.Errorf("failed to fetch API data: %w", err)
	}

	// Process raw data into rich domain models
	artists := r.processArtists(apiArtists, apiRelations)

	// Cache images if enabled
	var cachedCount, downloadedCount int
	if config.WithCache {
		cachedCount, downloadedCount = r.cacheImages(artists)
	}

	// Create location data from artist concert information
	locations := r.createLocations(artists)

	// Store processed data with pointer optimization
	r.loadProcessedData(artists, locations, cachedCount, downloadedCount)

	return nil
}

// GetArtists returns all artists as a slice of pointers.
func (r *Repository) GetArtists() []*Artist {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Return a copy to prevent external modification
	result := make([]*Artist, len(r.artists))
	copy(result, r.artists)
	return result
}

// GetArtistByID returns the artist with the given ID and true if found.
func (r *Repository) GetArtistByID(id int) (*Artist, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if index, exists := r.artistIndex[id]; exists {
		return r.artists[index], true
	}
	return nil, false
}

// GetArtistBySlug returns the artist with the given slug and true if found.
func (r *Repository) GetArtistBySlug(slug string) (*Artist, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if index, exists := r.artistSlugMap[slug]; exists {
		return r.artists[index], true
	}
	return nil, false
}

// GetLocations returns all locations as a slice of pointers.
func (r *Repository) GetLocations() []*Location {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]*Location, len(r.locations))
	copy(result, r.locations)
	return result
}

// GetLocationBySlug returns the location with the given slug and true if found.
func (r *Repository) GetLocationBySlug(slug string) (*Location, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if index, exists := r.locationIndex[slug]; exists {
		return r.locations[index], true
	}
	return nil, false
}

// GetStats returns basic statistics about the loaded data.
func (r *Repository) GetStats() map[string]int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return map[string]int{
		"artists":   len(r.artists),
		"locations": len(r.locations),
	}
}

// GetAppStats returns comprehensive application statistics.
func (r *Repository) GetAppStats() AppStats {
	r.mu.RLock()
	defer r.mu.RUnlock()

	stats := AppStats{
		TotalArtists:   len(r.artists),
		TotalLocations: len(r.locations),
	}

	// Calculate derived statistics
	totalMembers := 0
	totalConcerts := 0
	countrySet := make(map[string]bool)

	for _, artist := range r.artists {
		totalMembers += len(artist.Members)
		totalConcerts += artist.ConcertCount

		for _, country := range artist.Countries {
			countrySet[country] = true
		}
	}

	stats.TotalMembers = totalMembers
	stats.TotalConcerts = totalConcerts
	stats.TotalCountries = len(countrySet)

	return stats
}

// GetAdjacentArtists returns the previous and next artists in alphabetical order.
func (r *Repository) GetAdjacentArtists(currentID int) (prev, next *Artist) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	currentIndex, exists := r.artistIndex[currentID]
	if !exists || len(r.artists) == 0 {
		return nil, nil
	}

	if currentIndex > 0 {
		prev = r.artists[currentIndex-1]
	}

	if currentIndex < len(r.artists)-1 {
		next = r.artists[currentIndex+1]
	}

	return prev, next
}

// IsCacheEnabled returns whether image caching is enabled.
func (r *Repository) IsCacheEnabled() bool {
	return config.WithCache
}

// SetTestData allows setting test data directly for testing purposes.
func (r *Repository) SetTestData(artists []Artist, locations []Location) {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Convert to pointer slices
	r.artists = make([]*Artist, len(artists))
	for i := range artists {
		r.artists[i] = &artists[i]
	}

	r.locations = make([]*Location, len(locations))
	for i := range locations {
		r.locations[i] = &locations[i]
	}

	// Build artist index
	r.artistIndex = make(map[int]int, len(r.artists))
	for i, artist := range r.artists {
		r.artistIndex[artist.ID] = i
	}
}

// --- Private Helper Methods ---

// fetchAPIData loads data from all required API endpoints.
func (r *Repository) fetchAPIData(ctx context.Context) ([]APIArtist, APIRelation, error) {
	var apiArtists []APIArtist
	var apiRelations APIRelation

	// Fetch artists data
	if err := r.fetchJSON(ctx, "/artists", &apiArtists); err != nil {
		return nil, apiRelations, fmt.Errorf("failed to fetch artists: %w", err)
	}

	// Fetch relations data
	if err := r.fetchJSON(ctx, "/relation", &apiRelations); err != nil {
		return nil, apiRelations, fmt.Errorf("failed to fetch relations: %w", err)
	}

	return apiArtists, apiRelations, nil
}

// fetchJSON performs HTTP request and JSON unmarshaling for API endpoints.
func (r *Repository) fetchJSON(ctx context.Context, path string, dest any) error {
	url := config.APIBaseURL + path

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return err
	}

	client := &http.Client{
		Timeout: config.APIRequestTimeout,
	}

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("API returned status %d for %s", resp.StatusCode, path)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	return json.Unmarshal(body, dest)
}

// processArtists converts API data to rich domain models.
func (r *Repository) processArtists(apiArtists []APIArtist, apiRelations APIRelation) []Artist {
	artists := r.transformAPIArtists(apiArtists)
	return r.addConcertData(artists, apiRelations)
}

// transformAPIArtists converts API artist data to domain Artist objects.
func (r *Repository) transformAPIArtists(apiArtists []APIArtist) []Artist {
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
			Concerts:        []Concert{},
			DatesAtLocation: make(map[string][]string),
			ConcertCount:    0,
			Countries:       []string{},
		}
		artists = append(artists, artist)
	}

	return artists
}

// addConcertData enriches artists with concert information from relations API.
func (r *Repository) addConcertData(artists []Artist, apiRelations APIRelation) []Artist {
	// Create lookup map for efficient artist access
	artistMap := make(map[int]*Artist)
	for i := range artists {
		artistMap[artists[i].ID] = &artists[i]
	}

	// Process each artist's concert data
	for _, relation := range apiRelations.Index {
		artist, exists := artistMap[relation.ID]
		if !exists {
			continue
		}

		var concerts []Concert
		countrySet := make(map[string]bool)

		// Process each location and its concert dates
		for location, dates := range relation.DatesLocations {
			// Store dates by location for fast lookup
			artist.DatesAtLocation[location] = dates

			// Extract country from location string
			if country := extractCountryFromLocation(location); country != "" {
				countrySet[country] = true
			}

			// Create concert records
			for _, date := range dates {
				concerts = append(concerts, Concert{
					Date:     date,
					Location: location,
				})
			}
		}

		artist.Concerts = concerts
		artist.ConcertCount = len(concerts)
		artist.Countries = r.convertCountriesMapToSlice(countrySet)
	}

	return artists
}

// convertCountriesMapToSlice converts a country set to a sorted slice.
func (r *Repository) convertCountriesMapToSlice(countriesMap map[string]bool) []string {
	countries := make([]string, 0, len(countriesMap))
	for country := range countriesMap {
		countries = append(countries, country)
	}
	sort.Strings(countries)
	return countries
}

// cacheImages downloads and caches artist images locally if caching is enabled.
func (r *Repository) cacheImages(artists []Artist) (cached, downloaded int) {
	cacheDir := "static/img/artists"
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return 0, 0
	}

	for _, artist := range artists {
		fileName := artist.Slug + ".jpg"
		filePath := filepath.Join(cacheDir, fileName)

		// Check if file already exists
		if _, err := os.Stat(filePath); err == nil {
			cached++
			continue
		}

		// Download image
		if r.downloadImage(artist.Image, filePath) {
			downloaded++
		}

		// Add small delay to be respectful to the image server
		time.Sleep(10 * time.Millisecond)
	}

	return cached, downloaded
}

// downloadImage downloads an image from URL and saves it to the specified path.
func (r *Repository) downloadImage(url, path string) bool {
	client := &http.Client{Timeout: 30 * time.Second}

	resp, err := client.Get(url)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return false
	}

	// Create the file
	file, err := os.Create(path)
	if err != nil {
		return false
	}
	defer file.Close()

	// Copy response body to file
	_, err = io.Copy(file, resp.Body)
	return err == nil
}

// createLocations builds location data from artist concert information.
func (r *Repository) createLocations(artists []Artist) []Location {
	locationMap := make(map[string]*Location)

	// Process all artist concerts to build location data
	for _, artist := range artists {
		for locationName, dates := range artist.DatesAtLocation {
			slug := createSlug(locationName)

			// Get or create location
			location, exists := locationMap[slug]
			if !exists {
				location = &Location{
					Name:          locationName,
					Slug:          slug,
					Artists:       []ArtistAtLocation{},
					ArtistCount:   0,
					TotalConcerts: 0,
					EarliestYear:  9999,
					LatestYear:    0,
				}
				locationMap[slug] = location
			}

			// Add artist to location
			concertCount := len(dates)
			location.Artists = append(location.Artists, ArtistAtLocation{
				Artist:       artist,
				ConcertCount: concertCount,
			})
			location.TotalConcerts += concertCount

			// Update year range
			for _, date := range dates {
				if year := extractYearFromDate(date); year > 0 {
					if year < location.EarliestYear {
						location.EarliestYear = year
					}
					if year > location.LatestYear {
						location.LatestYear = year
					}
				}
			}
		}
	}

	// Convert map to slice and compute final statistics
	locations := make([]Location, 0, len(locationMap))
	for _, location := range locationMap {
		location.ArtistCount = len(location.Artists)

		// Sort artists by concert count (descending)
		sort.Slice(location.Artists, func(i, j int) bool {
			return location.Artists[i].ConcertCount > location.Artists[j].ConcertCount
		})

		locations = append(locations, *location)
	}

	// Sort locations by name
	sort.Slice(locations, func(i, j int) bool {
		return locations[i].Name < locations[j].Name
	})

	return locations
}

// loadProcessedData stores the processed data with pointer optimization.
func (r *Repository) loadProcessedData(artists []Artist, locations []Location, cachedCount, downloadedCount int) {
	// Convert artists to pointer slice and sort alphabetically
	r.artists = make([]*Artist, len(artists))
	for i := range artists {
		r.artists[i] = &artists[i]
	}

	sort.Slice(r.artists, func(i, j int) bool {
		return r.artists[i].Name < r.artists[j].Name
	})

	// Build artist index for O(1) lookups
	r.artistIndex = make(map[int]int, len(r.artists))
	r.artistSlugMap = make(map[string]int, len(r.artists))
	for i, artist := range r.artists {
		r.artistIndex[artist.ID] = i
		r.artistSlugMap[artist.Slug] = i
	}

	// Convert locations to pointer slice and build location index
	r.locations = make([]*Location, len(locations))
	r.locationIndex = make(map[string]int, len(locations))
	for i := range locations {
		r.locations[i] = &locations[i]
		r.locationIndex[r.locations[i].Slug] = i
	}
}

// --- Utility Functions ---

// createSlug generates a URL-friendly slug from a name.
func createSlug(name string) string {
	// Convert to lowercase
	slug := strings.ToLower(name)

	// Replace spaces and special characters with hyphens
	replacer := strings.NewReplacer(
		" ", "-",
		"'", "",
		".", "",
		"&", "and",
		"/", "-",
		"(", "",
		")", "",
		",", "",
	)
	slug = replacer.Replace(slug)

	// Remove any remaining special characters and multiple hyphens
	var result strings.Builder
	prevHyphen := false

	for _, r := range slug {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			result.WriteRune(r)
			prevHyphen = false
		} else if !prevHyphen {
			result.WriteRune('-')
			prevHyphen = true
		}
	}

	// Trim leading/trailing hyphens
	return strings.Trim(result.String(), "-")
}

// extractCountryFromLocation extracts the country from a location string.
func extractCountryFromLocation(location string) string {
	parts := strings.Split(location, "-")
	if len(parts) >= 2 {
		return strings.ToUpper(parts[len(parts)-1])
	}
	return ""
}

// extractYearFromDate extracts the year from a date string.
func extractYearFromDate(dateStr string) int {
	// Match patterns like "01-01-2023", "2023-01-01", "01/01/23", etc.
	re := regexp.MustCompile(`\b(\d{4})\b|\b(\d{2})-(\d{2})-(\d{2})\b`)
	matches := re.FindStringSubmatch(dateStr)

	if len(matches) > 1 && matches[1] != "" {
		// Found 4-digit year
		var year int
		fmt.Sscanf(matches[1], "%d", &year)
		return year
	}

	if len(matches) > 4 && matches[4] != "" {
		// Found 2-digit year, assume 20xx
		var year int
		fmt.Sscanf(matches[4], "%d", &year)
		if year < 50 {
			return 2000 + year
		}
		return 1900 + year
	}

	return 0
}
