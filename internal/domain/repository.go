package domain

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"groupie-tracker/internal/api"
)

// Repository provides centralized data management for the Groupie Tracker application.
type Repository struct {
	// API client for external data fetching
	apiClient *api.Client

	// Configuration fields
	withCache bool // Controls whether image caching is enabled

	// Simple cache enabled flag replaces complex CacheStatus enum
	cacheEnabled bool // True if image caching is enabled and functional

	// Core data collections (loaded once, read-only after initialization)
	artists         []Artist            // All artists sorted by name
	artistsByID     map[int]Artist      // Fast artist lookup by ID
	artistsBySlug   map[string]Artist   // Fast artist lookup by URL slug
	locations       []Location          // All locations sorted by concert count (descending)
	locationsBySlug map[string]Location // Fast location lookup by URL slug
	globalStats     map[string]int      // Pre-computed application statistics (legacy)
	appStats        AppStats            // Type-safe application statistics
}

// NewRepository creates a new repository instance with the provided API client.
func NewRepository(apiClient *api.Client, withCache bool) *Repository {
	return &Repository{
		apiClient: apiClient,
		withCache: withCache,
	}
}

// LoadData orchestrates the complete data loading and processing pipeline.
func (r *Repository) LoadData(ctx context.Context) error {
	// Fetch raw data from API using injected client
	apiArtists, err := r.apiClient.FetchArtists(ctx)
	if err != nil {
		return fmt.Errorf("failed to fetch artists: %w", err)
	}

	apiRelations, err := r.apiClient.FetchRelations(ctx)
	if err != nil {
		return fmt.Errorf("failed to fetch relations: %w", err)
	}

	// Process artists with their concert data
	artists := r.processArtists(apiArtists, apiRelations)

	// Cache images if enabled and get statistics
	cachedCount, downloadedCount := r.cacheImages(artists)

	// Create locations from artist data
	locations := r.createLocations(artists)

	// Store processed data with cache statistics
	r.loadProcessedData(artists, locations, cachedCount, downloadedCount)

	return nil
}

// --- Public Data Access Methods ---
//
// These methods provide thread-safe read-only access to the loaded data.
// All methods return pre-processed, ready-to-use data structures.

// GetArtists returns the complete artist collection sorted alphabetically by name.
//
// The returned slice contains fully populated Artist objects with:
//   - Basic information (name, members, creation year, etc.)
//   - Concert data and derived statistics
//   - Navigation links (next/previous artist IDs)
//   - SEO-friendly slugs for URL generation
//
// This method is commonly used for artist listing pages and search operations.
func (r *Repository) GetArtists() []Artist {
	return r.artists
}

// GetArtistByID performs fast artist lookup by unique identifier.
//
// Uses an internal map index for O(1) lookup performance. The ID parameter
// should match the original API artist ID.
//
// Returns the complete Artist object and a boolean indicating if the artist was found.
func (r *Repository) GetArtistByID(id int) (Artist, bool) {
	artist, found := r.artistsByID[id]
	return artist, found
}

// GetArtistBySlug performs fast artist lookup by URL-friendly slug.
//
// Slugs are generated from artist names using URL-safe characters (e.g., "queen", "led-zeppelin").
// This enables SEO-friendly URLs like /artists/queen instead of /artists/28.
//
// Returns the complete Artist object and a boolean indicating if the slug was found.
func (r *Repository) GetArtistBySlug(slug string) (Artist, bool) {
	artist, found := r.artistsBySlug[slug]
	return artist, found
}

// GetLocations returns the complete location collection sorted by popularity.
//
// Locations are sorted by total concert count in descending order, so the most
// popular venues appear first. Each Location contains:
//   - Venue information and geographic data
//   - Artist performance statistics
//   - Concert date ranges and counts
//   - SEO-friendly slugs for URL generation
//
// This method is used for location listing pages and geographic analysis.
func (r *Repository) GetLocations() []Location {
	return r.locations
}

// GetLocationBySlug performs fast location lookup by URL-friendly slug.
//
// Slugs are generated from location names using URL-safe characters (e.g., "london-uk").
// This enables SEO-friendly URLs like /locations/london-uk.
//
// Returns the complete Location object and a boolean indicating if the slug was found.
func (r *Repository) GetLocationBySlug(slug string) (Location, bool) {
	location, found := r.locationsBySlug[slug]
	return location, found
}

// GetStats returns pre-computed global application statistics.
//
// The statistics map includes comprehensive metrics computed during data loading:
//   - "total_artists": Number of artists in the dataset
//   - "total_members": Sum of all band members across all artists
//   - "total_locations": Number of unique concert venues
//   - "total_concerts": Total number of concert events
//   - "total_countries": Number of unique countries with concerts
//   - "cached_images": Number of artist images served from local cache
//   - "downloaded_images": Number of artist images downloaded during this session
//
// These statistics are used for dashboard displays and system monitoring.
// The cache-related statistics help track image optimization effectiveness.
func (r *Repository) GetStats() map[string]int {
	return r.globalStats
}

// GetAppStats returns type-safe application statistics.
//
// This method provides the same statistical information as GetStats() but
// with compile-time type safety and better API documentation. Prefer using
// this method for new code while maintaining GetStats() for backward compatibility.
func (r *Repository) GetAppStats() AppStats {
	return r.appStats
}

// --- Private Data Processing Pipeline ---
//
// These methods implement the data loading and transformation pipeline.
// They are called internally by LoadData() and should not be used directly.

// processArtists transforms raw API data into enriched Artist domain models.
func (r *Repository) processArtists(apiArtists []api.Artist, apiRelations api.Relation) []Artist {
	artists := r.transformAPIArtists(apiArtists)
	artists = r.addConcertData(artists, apiRelations)
	return artists
}

// transformAPIArtists converts raw API artist data to domain Artist objects.
func (r *Repository) transformAPIArtists(apiArtists []api.Artist) []Artist {
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
		}
		artists = append(artists, artist)
	}

	return artists
}

// addConcertData enriches artists with concert information from API relations.
func (r *Repository) addConcertData(artists []Artist, apiRelations api.Relation) []Artist {
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
				countries[r.extractCountryFromLocation(normalizedLoc)] = true
			}

			// Sort concerts chronologically
			sort.Slice(artist.Concerts, func(i, j int) bool {
				return artist.Concerts[i].Date < artist.Concerts[j].Date
			})

			// Set derived fields
			artist.ConcertCount = len(artist.Concerts)
			artist.Countries = r.convertCountriesMapToSlice(countries)
		}
	}

	// Sort artists by name for consistent display
	sort.Slice(artists, func(i, j int) bool {
		return artists[i].Name < artists[j].Name
	})

	return artists
}

// convertCountriesMapToSlice converts a map[string]bool to sorted string slice.
// This creates a consistent ordered list of countries for the artist.
func (r *Repository) convertCountriesMapToSlice(countriesMap map[string]bool) []string {
	countries := make([]string, 0, len(countriesMap))
	for country := range countriesMap {
		if country != "" { // Skip empty countries
			countries = append(countries, country)
		}
	}
	sort.Strings(countries)
	return countries
}

// GetAdjacentArtists finds the previous and next artists in the collection
// based on alphabetical order by name. This replaces pre-computed navigation IDs
// with on-demand lookup to reduce memory usage and complexity.
func (r *Repository) GetAdjacentArtists(currentID int) (prev, next *Artist) {
	if len(r.artists) == 0 {
		return nil, nil
	}

	// Find the current artist index
	currentIndex := -1
	for i, artist := range r.artists {
		if artist.ID == currentID {
			currentIndex = i
			break
		}
	}

	if currentIndex == -1 {
		return nil, nil // Artist not found
	}

	// Get previous artist (if not first)
	if currentIndex > 0 {
		prev = &r.artists[currentIndex-1]
	}

	// Get next artist (if not last)
	if currentIndex < len(r.artists)-1 {
		next = &r.artists[currentIndex+1]
	}

	return prev, next
}

// IsCacheEnabled returns true if image caching is enabled and functional.
func (r *Repository) IsCacheEnabled() bool {
	return r.cacheEnabled
}

// cacheImages handles local image caching optimization for artist photos.
//
// When image caching is enabled (config.WithCache = true), this method:
//
// Cache Management:
//   - Creates local cache directory (static/img/artists/) if needed
//   - Checks for existing cached images to avoid re-downloading
//   - Downloads missing images from original URLs
//   - Updates Artist.Image URLs to point to local cached versions
//
// Performance Optimization:
//   - Reduces external HTTP requests during normal operation
//   - Improves page load times by serving images locally
//   - Tracks cache hit/miss statistics for monitoring
//
// Cache State Tracking:
//   - Sets cacheEnabled flag based on successful cache initialization
//   - Simplified boolean approach replaces complex status enum
//
// The method modifies Artist objects in-place to update image URLs. It gracefully
// handles download failures by leaving original URLs intact. Cache statistics
// are used for monitoring and dashboard display.
//
// Returns counts of cached and newly downloaded images for statistics.
// Returns counts of cached and newly downloaded images for statistics.
func (r *Repository) cacheImages(artists []Artist) (cached, downloaded int) {
	if !r.withCache {
		r.cacheEnabled = false
		return 0, 0
	}

	cacheDir := "static/img/artists"
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		r.cacheEnabled = false
		return 0, 0
	}

	r.cacheEnabled = true
	for i := range artists {
		artist := &artists[i]
		fileName := fmt.Sprintf("%s.jpg", artist.Slug)
		filePath := filepath.Join(cacheDir, fileName)
		localPath := "/" + filepath.ToSlash(filePath)

		// Use cached file if it exists
		if _, err := os.Stat(filePath); err == nil {
			artist.Image = localPath
			cached++
			continue
		}

		// Download image
		if r.downloadImage(artist.Image, filePath) {
			artist.Image = localPath
			downloaded++
		}
	}

	return cached, downloaded
}

// downloadImage downloads and saves a single image from a URL to local filesystem.
//
// This helper method handles the low-level image downloading process:
//   - Validates the source URL is not empty
//   - Creates HTTP GET request for the image
//   - Validates successful HTTP response (200 OK)
//
// downloadImage fetches an artist image and saves it to the local cache.
func (r *Repository) downloadImage(url, path string) bool {
	if strings.TrimSpace(url) == "" {
		return false
	}

	resp, err := http.Get(url)
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

// createLocations analyzes artist concert data to build comprehensive location models.
//
// This method performs venue analysis and aggregation:
//
// Venue Discovery:
//   - Extracts unique concert locations from all artist concert data
//   - Normalizes location names for consistent identification
//   - Creates SEO-friendly slugs for location URLs
//
// Statistical Analysis:
//   - Counts total concerts held at each venue
//   - Tracks which artists performed at each venue and how often
//   - Computes temporal ranges (earliest/latest concert years)
//   - Ranks venues by total concert activity
//
// Data Organization:
//   - Sorts artists within each venue by performance frequency
//   - Sorts all venues by total concerts (most active first)
//   - Pre-computes aggregate fields for efficient display
//
// Performance Optimization:
//   - Uses maps for efficient concert counting during processing
//   - Converts to final slice format for consistent ordering
//   - Enables fast venue-based queries without real-time aggregation
//
// The resulting Location objects contain complete venue analytics ready for
// location detail pages and geographic visualizations.
//
// Returns a slice of Location objects sorted by concert volume (descending).
func (r *Repository) createLocations(artists []Artist) []Location {
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
			year := r.extractYearFromDate(concert.Date)
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

// loadProcessedData stores all processed data in repository indexes and computes global statistics.
//
// This final step of the data loading pipeline:
//
// Index Creation:
//   - Builds fast lookup maps (by ID, by slug) for artists and locations
//   - Stores sorted slices for consistent ordering in listings
//   - Enables O(1) lookups for detail page requests
//
// Statistics Computation:
//   - Aggregates counts across all artists and locations
//   - Computes unique country count from artist concert data
//   - Includes image cache effectiveness metrics
//   - Prepares data for dashboard displays
//
// Memory Organization:
//   - All data is stored in repository instance fields
//   - No additional processing required after this method
//   - Ready for concurrent read access from multiple goroutines
//
// The global statistics include both business metrics (artist counts, concert counts)
// and system metrics (cache performance) for comprehensive monitoring.
//
// After this method completes, the repository is fully initialized and ready
// to serve application requests efficiently.
func (r *Repository) loadProcessedData(artists []Artist, locations []Location, cachedCount, downloadedCount int) {
	// Store artists
	r.artists = artists
	r.artistsByID = make(map[int]Artist, len(artists))
	r.artistsBySlug = make(map[string]Artist, len(artists))

	totalMembers := 0
	totalConcerts := 0
	countries := make(map[string]bool)

	for _, artist := range artists {
		r.artistsByID[artist.ID] = artist
		r.artistsBySlug[artist.Slug] = artist
		totalMembers += len(artist.Members)
		totalConcerts += artist.ConcertCount

		for _, country := range artist.Countries {
			countries[country] = true
		}
	}

	// Store locations
	r.locations = locations
	r.locationsBySlug = make(map[string]Location, len(locations))
	for _, location := range locations {
		r.locationsBySlug[location.Slug] = location
	}

	// Store global stats including cache statistics
	r.appStats = AppStats{
		TotalArtists:     len(artists),
		TotalMembers:     totalMembers,
		TotalLocations:   len(locations),
		TotalConcerts:    totalConcerts,
		TotalCountries:   len(countries),
		CachedImages:     cachedCount,
		DownloadedImages: downloadedCount,
	}
	// Maintain legacy map format for backward compatibility
	r.globalStats = r.appStats.ToMap()
}

// --- Test Helper Methods ---

// SetTestData allows tests to populate the repository with test data.
// This method is only intended for use in test files and bypasses
// the normal data loading pipeline.
func (r *Repository) SetTestData(artists []Artist, locations []Location) {
	r.artists = artists
	r.locations = locations

	// Build indexes
	r.artistsByID = make(map[int]Artist)
	r.artistsBySlug = make(map[string]Artist)
	for _, artist := range artists {
		r.artistsByID[artist.ID] = artist
		r.artistsBySlug[artist.Slug] = artist
	}

	r.locationsBySlug = make(map[string]Location)
	for _, location := range locations {
		r.locationsBySlug[location.Slug] = location
	}

	// Mock stats (type-safe version)
	r.appStats = AppStats{
		TotalArtists:     len(artists),
		TotalMembers:     0,
		TotalLocations:   len(locations),
		TotalConcerts:    0,
		TotalCountries:   0,
		CachedImages:     0,
		DownloadedImages: 0,
	}
	// Maintain legacy map format for backward compatibility
	r.globalStats = r.appStats.ToMap()
}

// --- String Processing Utilities ---
//
// These helper functions provide consistent text transformation across the application.

// createSlug converts display names into URL-friendly identifiers.
//
// The slug generation process:
//  1. Converts name to lowercase for case-insensitive URLs
//  2. Replaces non-alphanumeric characters with hyphens
//  3. Removes leading/trailing hyphens for clean URLs
//  4. Handles special characters and spaces consistently
//
// Examples:
//   - "Queen" → "queen"
//   - "Led Zeppelin" → "led-zeppelin"
//   - "AC/DC" → "ac-dc"
//   - "Guns N' Roses" → "guns-n-roses"
//
// Used for both artist and location URL generation to maintain consistency.
func createSlug(name string) string {
	reg := regexp.MustCompile(`[^a-z0-9]+`)
	slug := reg.ReplaceAllString(strings.ToLower(name), "-")
	return strings.Trim(slug, "-")
}

// normalizeLocation converts raw API location strings to consistent internal format.
//
// The normalization process:
//   - Removes leading/trailing whitespace
//   - Converts to lowercase for consistent processing
//   - Replaces underscores with hyphens for URL compatibility
//   - Maintains original location structure (city-state-country)
//
// This ensures consistent location identification across the application,
// regardless of minor formatting variations in the source API data.
//
// Example: "New_York-USA" → "new-york-usa"
func normalizeLocation(location string) string {
	return strings.ToLower(strings.ReplaceAll(strings.TrimSpace(location), "_", "-"))
}
