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
	"strconv"
	"strings"

	"groupie-tracker/internal/config"
)

// Repository manages all application data and provides thread-safe access to it.
type Repository struct {
	// Configuration
	apiEndpoint string
	apiClient   *http.Client
	// Controls whether image caching is enabled
	withCache bool

	// CacheStatus indicates the state of the image cache after loading.
	CacheStatus CacheStatus

	// Pre-computed and sorted data collections
	artists         []Artist
	artistsByID     map[int]Artist
	artistsBySlug   map[string]Artist
	locations       []Location
	locationsBySlug map[string]Location
	globalStats     map[string]int
}

// NewRepository creates a new repository instance using values from the
// `internal/config` package. This keeps configuration centralized and makes
// tests simpler by allowing them to override `config` variables.
func NewRepository() *Repository {
	return &Repository{
		apiEndpoint: config.APIBaseURL,
		apiClient: &http.Client{
			Timeout: config.APIRequestTimeout,
		},
		withCache: config.WithCache,
	}
}

// LoadData fetches and processes all application data in a simple, sequential manner.
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

// --- Getters ---

// GetArtists returns a pre-sorted slice of all artists.
func (r *Repository) GetArtists() []Artist {
	return r.artists
}

// GetArtistByID returns an artist by their ID.
func (r *Repository) GetArtistByID(id int) (Artist, bool) {
	artist, found := r.artistsByID[id]
	return artist, found
}

// GetArtistBySlug returns an artist by their slug.
func (r *Repository) GetArtistBySlug(slug string) (Artist, bool) {
	artist, found := r.artistsBySlug[slug]
	return artist, found
}

// GetLocations returns a pre-sorted slice of all locations.
func (r *Repository) GetLocations() []Location {
	return r.locations
}

// GetLocationBySlug returns a location by its slug.
func (r *Repository) GetLocationBySlug(slug string) (Location, bool) {
	location, found := r.locationsBySlug[slug]
	return location, found
}

// GetStats returns pre-computed global statistics.
func (r *Repository) GetStats() map[string]int {
	return r.globalStats
}

// --- Private Helper Methods ---

// fetchAPIData retrieves raw API payloads needed for repository initialization.
// Returns the full set of artists and relations straight from the API.
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

// fetchJSON performs a GET request against the repository's baseURL+path and
// decodes the JSON response into dest. It returns a wrapped error with context
// if any network or decoding issues occur.
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

// processArtists converts API data to internal Artist models with enriched data.
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

// cacheImages handles image caching when enabled and returns cache statistics.
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

// downloadImage downloads an image from URL to local path.
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

// createLocations builds location data from artist concerts.
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
					Name:    concert.Location,
					Slug:    createSlug(concert.Location),
					Artists: make([]ArtistAtLocation, 0),
				}
				artistConcertCount[concert.Location] = make(map[int]int)
			}

			// Count concerts per artist per location
			artistConcertCount[concert.Location][artist.ID]++
			locationMap[concert.Location].TotalConcerts++
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

// findArtistByID is a helper function to find an artist in a slice by ID.
func (r *Repository) findArtistByID(artists []Artist, id int) (Artist, bool) {
	for _, artist := range artists {
		if artist.ID == id {
			return artist, true
		}
	}
	return Artist{}, false
}

// loadProcessedData stores the processed data in repository indexes.
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

func createSlug(name string) string {
	reg := regexp.MustCompile(`[^a-z0-9]+`)
	slug := reg.ReplaceAllString(strings.ToLower(name), "-")
	return strings.Trim(slug, "-")
}

func normalizeLocation(location string) string {
	return strings.ToLower(strings.ReplaceAll(strings.TrimSpace(location), "_", "-"))
}

// --- Filter Functionality ---

// extractCountryFromLocation extracts the country from a location string
// Assumes location format like "city-state-country" or "city-country"
func (r *Repository) extractCountryFromLocation(location string) string {
	parts := strings.Split(strings.ToLower(location), "-")
	if len(parts) == 0 {
		return ""
	}
	
	// The country is typically the last part
	country := strings.TrimSpace(parts[len(parts)-1])
	
	// Handle common abbreviations/normalizations
	switch country {
	case "usa", "us":
		return "USA"
	case "uk":
		return "UK"
	case "uae":
		return "UAE"
	default:
		// Capitalize first letter of each word
		words := strings.Fields(strings.ReplaceAll(country, "-", " "))
		for i, word := range words {
			if len(word) > 0 {
				words[i] = strings.ToUpper(word[:1]) + strings.ToLower(word[1:])
			}
		}
		return strings.Join(words, " ")
	}
}

// FilterArtists filters the artists based on the provided criteria
func (r *Repository) FilterArtists(params FilterParams) []Artist {
	var filtered []Artist

	for _, artist := range r.artists {
		if r.matchesFilters(artist, params) {
			filtered = append(filtered, artist)
		}
	}

	return filtered
}

// matchesFilters checks if an artist matches the filter criteria
func (r *Repository) matchesFilters(artist Artist, params FilterParams) bool {
	// Creation year range filter
	if params.CreationYearFrom != nil && artist.CreationYear < *params.CreationYearFrom {
		return false
	}
	if params.CreationYearTo != nil && artist.CreationYear > *params.CreationYearTo {
		return false
	}

	// First album year filter (extract year from date string)
	if params.FirstAlbumYearFrom != nil || params.FirstAlbumYearTo != nil {
		albumYear := r.extractYearFromDate(artist.FirstAlbum)
		if albumYear > 0 {
			if params.FirstAlbumYearFrom != nil && albumYear < *params.FirstAlbumYearFrom {
				return false
			}
			if params.FirstAlbumYearTo != nil && albumYear > *params.FirstAlbumYearTo {
				return false
			}
		}
	}

	// Member count checkbox filter - check if artist's member count is in the allowed list
	if len(params.MemberCounts) > 0 {
		memberCount := len(artist.Members)
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

	// Country filter - check if artist has concerts in any of the specified countries
	if len(params.Countries) > 0 {
		hasMatchingCountry := false
		artistCountries := make(map[string]bool)
		
		// Extract countries from artist's concert locations
		for _, concert := range artist.Concerts {
			country := r.extractCountryFromLocation(concert.Location)
			if country != "" {
				artistCountries[country] = true
			}
		}
		
		// Check if any artist country matches filter countries
		for _, filterCountry := range params.Countries {
			if artistCountries[filterCountry] {
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

// extractYearFromDate extracts year from various date formats
func (r *Repository) extractYearFromDate(dateStr string) int {
	// Handle common date formats
	if len(dateStr) >= 4 {
		// Check for YYYY at the end (DD-MM-YYYY)
		if len(dateStr) >= 10 && dateStr[2] == '-' && dateStr[5] == '-' {
			if year, err := strconv.Atoi(dateStr[6:10]); err == nil {
				return year
			}
		}
		// Check for YYYY at the beginning (YYYY-MM-DD or just YYYY)
		if year, err := strconv.Atoi(dateStr[:4]); err == nil && year > 1900 && year < 3000 {
			return year
		}
	}
	return 0
}

// GetFilterOptions returns the available filter options based on current data
func (r *Repository) GetFilterOptions() FilterOptions {
	if len(r.artists) == 0 {
		return FilterOptions{}
	}

	// Initialize with first artist's values
	minCreationYear, maxCreationYear := r.artists[0].CreationYear, r.artists[0].CreationYear
	minFirstAlbumYear, maxFirstAlbumYear := 0, 0
	memberCountSet := make(map[int]bool)
	countrySet := make(map[string]bool)

	for _, artist := range r.artists {
		// Creation year range
		if artist.CreationYear < minCreationYear {
			minCreationYear = artist.CreationYear
		}
		if artist.CreationYear > maxCreationYear {
			maxCreationYear = artist.CreationYear
		}

		// First album year range
		albumYear := r.extractYearFromDate(artist.FirstAlbum)
		if albumYear > 0 {
			if minFirstAlbumYear == 0 || albumYear < minFirstAlbumYear {
				minFirstAlbumYear = albumYear
			}
			if albumYear > maxFirstAlbumYear {
				maxFirstAlbumYear = albumYear
			}
		}

		// Collect unique member counts
		memberCount := len(artist.Members)
		memberCountSet[memberCount] = true

		// Collect unique countries from concert locations
		for _, concert := range artist.Concerts {
			country := r.extractCountryFromLocation(concert.Location)
			if country != "" {
				countrySet[country] = true
			}
		}
	}

	// Convert member count set to sorted slice (1 to max)
	memberCounts := make([]int, 0, len(memberCountSet))
	for count := range memberCountSet {
		memberCounts = append(memberCounts, count)
	}
	sort.Ints(memberCounts)

	// Convert country set to sorted slice
	countries := make([]string, 0, len(countrySet))
	for country := range countrySet {
		countries = append(countries, country)
	}
	sort.Strings(countries)

	// Set default first album year range if no valid years found
	if minFirstAlbumYear == 0 {
		minFirstAlbumYear = minCreationYear
	}
	if maxFirstAlbumYear == 0 {
		maxFirstAlbumYear = maxCreationYear
	}

	return FilterOptions{
		CreationYearMin:   minCreationYear,
		CreationYearMax:   maxCreationYear,
		FirstAlbumYearMin: minFirstAlbumYear,
		FirstAlbumYearMax: maxFirstAlbumYear,
		MemberCounts:      memberCounts,
		Countries:         countries,
	}
}
