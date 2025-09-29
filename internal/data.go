package data

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"sort"
	"strings"
	"time"
)

// --- External API Data Structures ---

// APIArtist represents the raw artist data structure from the /api/artists endpoint.
type APIArtist struct {
	ID           int      `json:"id"`
	Name         string   `json:"name"`
	Members      []string `json:"members"`
	CreationYear int      `json:"creationDate"`
	FirstAlbum   string   `json:"firstAlbum"`
	Image        string   `json:"image"`
}

// APIRelationIndex represents a single artist's concert data from the /api/relation endpoint.
type APIRelationIndex struct {
	ID             int                 `json:"id"`
	DatesLocations map[string][]string `json:"datesLocations"`
}

// APIRelation wraps the complete concert relations dataset from the /api/relation endpoint.
type APIRelation struct {
	Index []APIRelationIndex `json:"index"`
}

// --- Core Domain Models ---

// Concert represents a single concert performance by an artist.
type Concert struct {
	Location string `json:"location"`
	Country  string `json:"country"`
	Date     string `json:"date"`
	Year     int    `json:"year"`
}

// Artist represents the complete internal model of a music artist/band.
type Artist struct {
	ID           int       `json:"id"`
	Name         string    `json:"name"`
	Slug         string    `json:"slug"`
	Members      []string  `json:"members"`
	CreationYear int       `json:"creation_year"`
	FirstAlbum   string    `json:"first_album"`
	Image        string    `json:"image"`
	Concerts     []Concert `json:"concerts"`
	Countries    []string  `json:"countries"`
	ConcertCount int       `json:"concert_count"`
}

// Location represents a concert venue with aggregated statistics.
type Location struct {
	Name          string    `json:"name"`
	Slug          string    `json:"slug"`
	Artists       []string  `json:"artists"`
	ArtistCount   int       `json:"artist_count"`
	TotalConcerts int       `json:"total_concerts"`
	EarliestYear  int       `json:"earliest_year"`
	LatestYear    int       `json:"latest_year"`
	Concerts      []Concert `json:"concerts"`
}

// AppStats provides type-safe application statistics.
type AppStats struct {
	TotalArtists     int `json:"total_artists"`
	TotalLocations   int `json:"total_locations"`
	TotalConcerts    int `json:"total_concerts"`
	EarliestYear     int `json:"earliest_year"`
	LatestYear       int `json:"latest_year"`
	CachedImages     int `json:"cached_images"`
	DownloadedImages int `json:"downloaded_images"`
}

// --- Simplified Data Store ---

// DataStore provides simple, centralized data management.
// Uses "load once, read many" pattern with thread-safe read operations.
type DataStore struct {
	Artists         []Artist            `json:"artists"`
	ArtistsByID     map[int]Artist      `json:"-"`
	ArtistsBySlug   map[string]Artist   `json:"-"`
	Locations       []Location          `json:"locations"`
	LocationsBySlug map[string]Location `json:"-"`
	Stats           AppStats            `json:"stats"`
}

// Global data store instance with strategic caching
var (
	store       *DataStore
	apiBaseURL  = "https://groupietrackers.herokuapp.com/api"
	httpTimeout = 10 * time.Second
)

// LoadData loads all data from the API and returns a populated DataStore.
// This replaces the complex Repository.LoadData method with a simple function.
func LoadData(ctx context.Context) (*DataStore, error) {
	// Create HTTP client
	client := &http.Client{Timeout: httpTimeout}

	// Fetch data from API
	apiArtists, apiRelations, err := fetchAPIData(ctx, client)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch API data: %w", err)
	}

	// Process artists with concert data
	artists := processArtists(apiArtists, apiRelations)

	// Create locations from artist data
	locations := createLocations(artists)

	// Build indexes for fast lookups
	artistsByID := make(map[int]Artist)
	artistsBySlug := make(map[string]Artist)
	for _, artist := range artists {
		artistsByID[artist.ID] = artist
		artistsBySlug[artist.Slug] = artist
	}

	locationsBySlug := make(map[string]Location)
	for _, location := range locations {
		locationsBySlug[location.Slug] = location
	}

	// Calculate statistics
	stats := calculateStats(artists, locations)

	// Create and return data store
	dataStore := &DataStore{
		Artists:         artists,
		ArtistsByID:     artistsByID,
		ArtistsBySlug:   artistsBySlug,
		Locations:       locations,
		LocationsBySlug: locationsBySlug,
		Stats:           stats,
	}

	// Cache globally for package-level access
	store = dataStore

	return dataStore, nil
}

// GetDataStore returns the globally cached data store.
// Must call LoadData first.
func GetDataStore() *DataStore {
	return store
}

// --- Data Loading Functions ---

// fetchAPIData retrieves raw data from the Groupie Tracker API.
func fetchAPIData(ctx context.Context, client *http.Client) ([]APIArtist, []APIRelationIndex, error) {
	// Fetch artists
	var artists []APIArtist
	if err := fetchJSONEndpoint(ctx, client, apiBaseURL+"/artists", &artists); err != nil {
		return nil, nil, fmt.Errorf("failed to fetch artists: %w", err)
	}

	// Fetch relations
	var relations APIRelation
	if err := fetchJSONEndpoint(ctx, client, apiBaseURL+"/relation", &relations); err != nil {
		return nil, nil, fmt.Errorf("failed to fetch relations: %w", err)
	}

	return artists, relations.Index, nil
}

// fetchJSONEndpoint makes an HTTP request and decodes JSON response.
func fetchJSONEndpoint(ctx context.Context, client *http.Client, url string, target interface{}) error {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return err
	}

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	return json.Unmarshal(body, target)
}

// processArtists converts API data to domain models with enriched data.
func processArtists(apiArtists []APIArtist, apiRelations []APIRelationIndex) []Artist {
	// Create relation map for fast lookup
	relationMap := make(map[int]APIRelationIndex)
	for _, relation := range apiRelations {
		relationMap[relation.ID] = relation
	}

	var artists []Artist
	for _, apiArtist := range apiArtists {
		artist := Artist{
			ID:           apiArtist.ID,
			Name:         apiArtist.Name,
			Slug:         generateSlug(apiArtist.Name),
			Members:      apiArtist.Members,
			CreationYear: apiArtist.CreationYear,
			FirstAlbum:   apiArtist.FirstAlbum,
			Image:        apiArtist.Image,
		}

		// Add concert data if available
		if relation, exists := relationMap[apiArtist.ID]; exists {
			concerts, countries := processConcerts(relation.DatesLocations)
			artist.Concerts = concerts
			artist.Countries = countries
			artist.ConcertCount = len(concerts)
		}

		artists = append(artists, artist)
	}

	// Sort artists by name
	sort.Slice(artists, func(i, j int) bool {
		return artists[i].Name < artists[j].Name
	})

	return artists
}

// processConcerts converts API concert data to domain models.
func processConcerts(datesLocations map[string][]string) ([]Concert, []string) {
	var concerts []Concert
	countrySet := make(map[string]bool)

	for location, dates := range datesLocations {
		country := extractCountryFromLocation(location)
		if country != "" {
			countrySet[country] = true
		}

		for _, date := range dates {
			concert := Concert{
				Location: location,
				Country:  country,
				Date:     date,
				Year:     extractYearFromDate(date),
			}
			concerts = append(concerts, concert)
		}
	}

	// Convert country set to sorted slice
	var countries []string
	for country := range countrySet {
		countries = append(countries, country)
	}
	sort.Strings(countries)

	return concerts, countries
}

// createLocations generates location data from artist concert information.
func createLocations(artists []Artist) []Location {
	locationMap := make(map[string]*Location)

	// Aggregate data by location
	for _, artist := range artists {
		for _, concert := range artist.Concerts {
			if concert.Location == "" {
				continue
			}

			if loc, exists := locationMap[concert.Location]; exists {
				// Update existing location
				loc.TotalConcerts++
				if concert.Year > 0 {
					if loc.EarliestYear == 0 || concert.Year < loc.EarliestYear {
						loc.EarliestYear = concert.Year
					}
					if concert.Year > loc.LatestYear {
						loc.LatestYear = concert.Year
					}
				}

				// Add artist if not already present
				artistExists := false
				for _, existingArtist := range loc.Artists {
					if existingArtist == artist.Name {
						artistExists = true
						break
					}
				}
				if !artistExists {
					loc.Artists = append(loc.Artists, artist.Name)
					loc.ArtistCount++
				}

				loc.Concerts = append(loc.Concerts, concert)
			} else {
				// Create new location
				location := &Location{
					Name:          concert.Location,
					Slug:          generateSlug(concert.Location),
					Artists:       []string{artist.Name},
					ArtistCount:   1,
					TotalConcerts: 1,
					Concerts:      []Concert{concert},
				}

				if concert.Year > 0 {
					location.EarliestYear = concert.Year
					location.LatestYear = concert.Year
				}

				locationMap[concert.Location] = location
			}
		}
	}

	// Convert map to slice and sort by concert count (descending)
	var locations []Location
	for _, location := range locationMap {
		locations = append(locations, *location)
	}

	sort.Slice(locations, func(i, j int) bool {
		return locations[i].TotalConcerts > locations[j].TotalConcerts
	})

	return locations
}

// calculateStats computes application-wide statistics.
func calculateStats(artists []Artist, locations []Location) AppStats {
	stats := AppStats{
		TotalArtists:   len(artists),
		TotalLocations: len(locations),
	}

	for _, artist := range artists {
		stats.TotalConcerts += artist.ConcertCount

		// Track earliest/latest years from concerts
		for _, concert := range artist.Concerts {
			if concert.Year > 0 {
				if stats.EarliestYear == 0 || concert.Year < stats.EarliestYear {
					stats.EarliestYear = concert.Year
				}
				if concert.Year > stats.LatestYear {
					stats.LatestYear = concert.Year
				}
			}
		}
	}

	return stats
}

// --- Utility Functions ---

// generateSlug creates a URL-friendly slug from a string.
func generateSlug(s string) string {
	// Convert to lowercase
	slug := strings.ToLower(s)

	// Replace spaces and special characters with hyphens
	reg := regexp.MustCompile(`[^a-z0-9]+`)
	slug = reg.ReplaceAllString(slug, "-")

	// Remove leading/trailing hyphens
	slug = strings.Trim(slug, "-")

	return slug
}

// extractCountryFromLocation parses location strings to extract country names.
func extractCountryFromLocation(location string) string {
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

// extractYearFromDate parses various date string formats to extract calendar years.
func extractYearFromDate(dateStr string) int {
	// Handle common date formats
	if len(dateStr) >= 4 {
		// Check for YYYY at the end (DD-MM-YYYY)
		if len(dateStr) >= 10 && dateStr[2] == '-' && dateStr[5] == '-' {
			if year := parseYear(dateStr[6:10]); year > 0 {
				return year
			}
		}
		// Check for YYYY at the beginning (YYYY-MM-DD or just YYYY)
		if year := parseYear(dateStr[:4]); year > 0 {
			return year
		}
	}
	return 0
}

// parseYear safely parses a year string.
func parseYear(yearStr string) int {
	if len(yearStr) != 4 {
		return 0
	}

	year := 0
	for _, ch := range yearStr {
		if ch < '0' || ch > '9' {
			return 0
		}
		year = year*10 + int(ch-'0')
	}

	if year > 1900 && year < 3000 {
		return year
	}
	return 0
}
