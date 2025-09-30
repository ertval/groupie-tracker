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

	"groupie-tracker/internal/config"
)

// Repository provides centralized data management for the Groupie Tracker application.
//
// This is the core data access layer that coordinates all data operations:
//
// Data Loading:
//   - Fetches raw data from the external Groupie Tracker API
//   - Transforms API responses into rich domain models
//   - Builds searchable indexes for efficient lookups
//
// Performance Optimization:
//   - Pre-computes derived fields (concert counts, country lists, etc.)
//   - Creates slug-based indexes for SEO-friendly URLs
//   - Optionally caches artist images locally to reduce external requests
//
// Thread Safety:
//   - All data is loaded once during initialization
//   - After LoadData(), all access methods are read-only and thread-safe
//   - No mutating operations allowed after initial load
//
// The repository follows a "load once, read many" pattern that optimizes
// for the typical web application access pattern.
type Repository struct {
	// Configuration fields
	apiEndpoint string       // Base URL for the Groupie Tracker API
	apiClient   *http.Client // HTTP client with configured timeout
	withCache   bool         // Controls whether image caching is enabled

	// CacheStatus indicates the state of the image cache after the most recent data load
	CacheStatus CacheStatus

	// Core data collections (loaded once, read-only after initialization)
	artists         []Artist            // All artists sorted by name
	artistsByID     map[int]Artist      // Fast artist lookup by ID
	artistsBySlug   map[string]Artist   // Fast artist lookup by URL slug
	locations       []Location          // All locations sorted by concert count (descending)
	locationsBySlug map[string]Location // Fast location lookup by URL slug
	globalStats     map[string]int      // Pre-computed application statistics
}

// NewRepository creates a new repository instance configured from the global config package.
//
// This constructor uses values from `internal/config` rather than accepting parameters,
// which centralizes configuration management and simplifies testing. Tests can modify
// config package variables directly to customize repository behavior.
//
// The returned repository is ready for data loading via LoadData().
func NewRepository() *Repository {
	return &Repository{
		apiEndpoint: config.APIBaseURL,
		apiClient: &http.Client{
			Timeout: config.APIRequestTimeout,
		},
		withCache: config.WithCache,
	}
}

// LoadData orchestrates the complete data loading and processing pipeline.
//
// This method performs all necessary operations to prepare the application for serving requests:
//
// 1. API Data Fetching:
//   - Retrieves artist records from /api/artists endpoint
//   - Retrieves concert relations from /api/relation endpoint
//   - Handles network errors and API response validation
//
// 2. Data Processing:
//   - Transforms API models to rich domain models
//   - Cross-references artist and relation data
//   - Computes derived fields (concert counts, country lists, navigation links)
//
// 3. Performance Optimization:
//   - Optionally downloads and caches artist images locally
//   - Builds fast lookup indexes (by ID, by slug)
//   - Pre-computes global statistics for dashboard display
//
// 4. Location Analysis:
//   - Extracts unique concert venues from artist data
//   - Aggregates venue statistics (concert counts, artist counts, date ranges)
//   - Sorts venues by popularity for location browsing
//
// After LoadData() completes successfully, all repository getter methods are
// ready to serve data efficiently. The method is designed to be called once
// during application startup.
//
// Returns an error if any part of the pipeline fails. The repository should
// not be used if LoadData() returns an error.
func (r *Repository) LoadData(ctx context.Context) error {
	// Fetch raw data from API
	apiArtists, apiRelations, err := r.fetchAPIData(ctx)
	if err != nil {
		return err
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

// --- Private Data Processing Pipeline ---
//
// These methods implement the data loading and transformation pipeline.
// They are called internally by LoadData() and should not be used directly.

// fetchAPIData retrieves raw JSON data from the external Groupie Tracker API endpoints.
//
// This method coordinates parallel fetching of two API endpoints:
//   - /api/artists: Complete artist information (name, members, creation year, etc.)
//   - /api/relation: Concert location and date mappings for all artists
//
// The method uses the repository's configured HTTP client with timeout to prevent
// hanging requests. It validates HTTP status codes and JSON response format.
//
// Returns the complete API datasets ready for processing, or an error if any
// network or parsing issues occur. Both endpoints must succeed for the method to succeed.
func (r *Repository) fetchAPIData(ctx context.Context) ([]APIArtist, APIRelation, error) {
	var apiArtists []APIArtist
	if err := r.fetchJSON(ctx, "/api/artists", &apiArtists); err != nil {
		return nil, APIRelation{}, fmt.Errorf("failed to fetch artists: %w", err)
	}

	var apiRelations APIRelation
	if err := r.fetchJSON(ctx, "/api/relation", &apiRelations); err != nil {
		return nil, APIRelation{}, fmt.Errorf("failed to fetch relations: %w", err)
	}
	return apiArtists, apiRelations, nil
}

// fetchJSON performs HTTP GET requests with JSON response parsing.
//
// This low-level helper method handles the standard API request pattern:
//  1. Creates HTTP request with context for cancellation support
//  2. Executes request using repository's configured HTTP client
//  3. Validates HTTP response status (must be 200 OK)
//  4. Parses JSON response into the provided destination struct
//
// The method provides consistent error handling and request patterns for all
// API endpoints. Context cancellation is supported for timeout handling.
//
// Parameters:
//   - ctx: Request context for cancellation and timeout
//   - path: API endpoint path (e.g., "/api/artists")
//   - dest: Pointer to struct for JSON unmarshaling
//
// Returns an error if any step fails (network, HTTP status, or JSON parsing).
func (r *Repository) fetchJSON(ctx context.Context, path string, dest any) error {
	req, err := http.NewRequestWithContext(ctx, "GET", r.apiEndpoint+path, nil)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}

	resp, err := r.apiClient.Do(req)
	if err != nil {
		return fmt.Errorf("making request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	if err := json.NewDecoder(resp.Body).Decode(dest); err != nil {
		return fmt.Errorf("decoding response: %w", err)
	}

	return nil
}

// processArtists transforms raw API data into enriched Artist domain models.
//
// This method performs the core business logic transformation:
//
// Data Integration:
//   - Merges artist records with concert relation data by ID
//   - Creates structured Concert objects from raw date/location mappings
//   - Handles missing or incomplete relation data gracefully
//
// Enrichment:
//   - Generates SEO-friendly URL slugs from artist names
//   - Extracts and normalizes country lists from concert locations
//   - Computes aggregate statistics (total concerts, countries)
//   - Creates fast-lookup maps for location-based queries
//
// Organization:
//   - Sorts concerts chronologically for each artist
//   - Sorts artists alphabetically by name for consistent display
//   - Establishes navigation links (previous/next artist IDs)
//
// The resulting Artist objects are ready for direct use in templates and APIs
// without additional processing. Country extraction and slug generation follow
// the same patterns used throughout the application.
//
// Returns a complete slice of processed Artist objects sorted by name.
func (r *Repository) processArtists(apiArtists []APIArtist, apiRelations APIRelation) []Artist {
	artists := make([]Artist, 0, len(apiArtists))
	relationMap := make(map[int]APIRelationIndex)

	// Index relations by artist ID for efficient lookup
	for _, rel := range apiRelations.Index {
		relationMap[rel.ID] = rel
	}

	// Build artists with concert data
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

		// Add concert data if available
		if rel, exists := relationMap[apiArtist.ID]; exists {
			countries := make(map[string]bool)

			for location, dates := range rel.DatesLocations {
				normalizedLoc := normalizeLocation(location)
				locationSlug := createSlug(normalizedLoc)
				artist.DatesAtLocation[locationSlug] = dates

				for _, date := range dates {
					artist.Concerts = append(artist.Concerts, Concert{
						Date:     date,
						Location: normalizedLoc,
					})
				}

				// Extract countries
				parts := strings.Split(normalizedLoc, "-")
				if len(parts) > 1 {
					country := strings.TrimSpace(parts[len(parts)-1])
					countries[country] = true
				}
			}

			// Sort concerts by date
			sort.Slice(artist.Concerts, func(i, j int) bool {
				return artist.Concerts[i].Date < artist.Concerts[j].Date
			})

			// Set derived fields
			artist.ConcertCount = len(artist.Concerts)
			artist.Countries = make([]string, 0, len(countries))
			for country := range countries {
				artist.Countries = append(artist.Countries, country)
			}
			sort.Strings(artist.Countries)
		}

		artists = append(artists, artist)
	}

	// Sort artists by name and set navigation links
	sort.Slice(artists, func(i, j int) bool {
		return artists[i].Name < artists[j].Name
	})

	for i := range artists {
		if i > 0 {
			artists[i].PrevArtistID = artists[i-1].ID
		}
		if i < len(artists)-1 {
			artists[i].NextArtistID = artists[i+1].ID
		}
	}

	return artists
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
//   - Sets CacheStatus based on cache effectiveness
//   - CacheWarm: All images served from existing cache
//   - CacheCold: Some or all images required downloading
//   - CacheDisabled: Caching is turned off
//
// The method modifies Artist objects in-place to update image URLs. It gracefully
// handles download failures by leaving original URLs intact. Cache statistics
// are used for monitoring and dashboard display.
//
// Returns counts of cached and newly downloaded images for statistics.
func (r *Repository) cacheImages(artists []Artist) (cached, downloaded int) {
	if !r.withCache {
		r.CacheStatus = CacheDisabled
		return 0, 0
	}

	cacheDir := "static/img/artists"
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		r.CacheStatus = CacheCold
		return 0, 0
	}

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

	if downloaded > 0 {
		r.CacheStatus = CacheCold
	} else {
		r.CacheStatus = CacheWarm
	}

	return cached, downloaded
}

// downloadImage downloads and saves a single image from a URL to local filesystem.
//
// This helper method handles the low-level image downloading process:
//   - Validates the source URL is not empty
//   - Creates HTTP GET request for the image
//   - Validates successful HTTP response (200 OK)
//   - Streams image data directly to local file
//   - Handles all error conditions gracefully
//
// The method uses streaming copy to handle large images efficiently without
// loading entire image into memory. It returns boolean success status rather
// than errors to simplify cache management logic.
//
// Returns true if image was successfully downloaded and saved, false otherwise.
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
			// Find the artist by ID
			if artist, found := r.findArtistByID(artists, artistID); found {
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

// findArtistByID performs linear search to locate an artist by ID within a slice.
//
// This helper method is used during location creation when cross-referencing
// artist IDs from concert data with the full artist objects. The method uses
// simple iteration since it's only called during initial data processing.
//
// Returns the complete Artist object and a boolean indicating if the ID was found.
func (r *Repository) findArtistByID(artists []Artist, id int) (Artist, bool) {
	for _, artist := range artists {
		if artist.ID == id {
			return artist, true
		}
	}
	return Artist{}, false
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
	r.globalStats = map[string]int{
		"total_artists":     len(artists),
		"total_members":     totalMembers,
		"total_locations":   len(locations),
		"total_concerts":    totalConcerts,
		"total_countries":   len(countries),
		"cached_images":     cachedCount,
		"downloaded_images": downloadedCount,
	}
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
