// Package data provides application data models and repository for the Groupie Tracker application.
package data

import (
	"context"
	"fmt"
	"log"
	"regexp"
	"sort"
	"strings"
)

// -----------------------------
// Application Data Models
// -----------------------------

// Artist represents an artist in the application domain
type Artist struct {
	ID           int
	Name         string
	Members      []string
	CreationYear int
	FirstAlbum   string
	Image        string
	Slug         string // Precomputed SEO-friendly URL slug
}

// Relation represents the concert relationship data in application domain
type Relation struct {
	ID             int
	DatesLocations map[string][]string
}

// LocationStat represents statistics for a location (precomputed)
type LocationStat struct {
	Name         string
	DisplayName  string
	Slug         string
	ArtistCount  int
	ConcertCount int
	Artists      []Artist
	Dates        []string
}

// ArtistWithDates pairs an artist with their concert dates at a location
type ArtistWithDates struct {
	Artist Artist
	Dates  []string
}

// LocationDetail provides detailed information about a location
type LocationDetail struct {
	Name             string
	DisplayName      string
	Slug             string
	Artists          []Artist
	ArtistsWithDates []ArtistWithDates
	Dates            []string
	ArtistCount      int
	ConcertCount     int
}

// PageData represents common data needed for all pages
type PageData struct {
	Title    string
	ExtraCSS string
	ExtraJS  string
}

// -----------------------------
// API Interface and Types
// -----------------------------

// APIClient defines the interface for fetching data from external APIs
type APIClient interface {
	FetchAllData(ctx context.Context) (*APIResponse, error)
}

// APIArtist represents an artist from the API response
type APIArtist struct {
	ID           int      `json:"id"`
	Image        string   `json:"image"`
	Name         string   `json:"name"`
	Members      []string `json:"members"`
	CreationYear int      `json:"creationDate"`
	FirstAlbum   string   `json:"firstAlbum"`
}

// APIRelation represents a relation from the API response
type APIRelation struct {
	ID             int                 `json:"id"`
	DatesLocations map[string][]string `json:"datesLocations"`
}

// APIResponse represents the complete API response structure
type APIResponse struct {
	Artists   []APIArtist   `json:"artists,omitempty"`
	Relations []APIRelation `json:"relations,omitempty"`
}

// -----------------------------
// Repository (Single Store)
// -----------------------------

// Repository provides unified data storage and retrieval for the application
type Repository struct {
	// Core data
	artists   map[int]Artist
	relations map[int]Relation

	// Precomputed indexes for performance
	artistSlugs     map[string]int // slug -> artist ID
	uniqueLocations []string       // cached unique locations
	uniqueDates     []string       // cached unique dates
	locationStats   []LocationStat // precomputed location statistics
	stats           map[string]int // precomputed global stats
}

// NewRepository creates a new empty repository
func NewRepository() *Repository {
	return &Repository{
		artists:         make(map[int]Artist),
		relations:       make(map[int]Relation),
		artistSlugs:     make(map[string]int),
		uniqueLocations: make([]string, 0),
		uniqueDates:     make([]string, 0),
		locationStats:   make([]LocationStat, 0),
		stats:           make(map[string]int),
	}
}

// -----------------------------
// Data Loading & Conversion
// -----------------------------

// InitializeWithAPI loads data from the API client and converts to application models
func (r *Repository) InitializeWithAPI(ctx context.Context, apiClient APIClient) error {
	apiData, err := apiClient.FetchAllData(ctx)
	if err != nil {
		return fmt.Errorf("failed to fetch data from API: %w", err)
	}

	r.LoadData(*apiData)
	return nil
}

// LoadData converts API data to application models and precomputes indexes
func (r *Repository) LoadData(apiData APIResponse) {
	// Clear existing data
	r.artists = make(map[int]Artist)
	r.relations = make(map[int]Relation)
	r.artistSlugs = make(map[string]int)

	// Convert API artists to application artists
	for _, apiArtist := range apiData.Artists {
		artist := Artist{
			ID:           apiArtist.ID,
			Name:         apiArtist.Name,
			Members:      apiArtist.Members,
			CreationYear: apiArtist.CreationYear,
			FirstAlbum:   apiArtist.FirstAlbum,
			Image:        apiArtist.Image,
			Slug:         generateArtistSlug(apiArtist.Name),
		}

		r.artists[artist.ID] = artist
		r.artistSlugs[artist.Slug] = artist.ID
	}

	// Convert API relations to application relations
	for _, apiRelation := range apiData.Relations {
		relation := Relation{
			ID:             apiRelation.ID,
			DatesLocations: apiRelation.DatesLocations,
		}
		r.relations[relation.ID] = relation
	}

	// Precompute all derived data
	r.precomputeIndexes()

	log.Printf("✅ Repository loaded: %d artists, %d relations",
		len(r.artists), len(r.relations))
} // -----------------------------
// Core Data Access Methods
// -----------------------------

// GetAllArtists returns all artists
func (r *Repository) GetAllArtists() []Artist {
	artists := make([]Artist, 0, len(r.artists))
	for _, artist := range r.artists {
		artists = append(artists, artist)
	}
	return artists
}

// GetArtist retrieves an artist by ID
func (r *Repository) GetArtist(id int) (Artist, bool) {
	artist, exists := r.artists[id]
	return artist, exists
}

// GetArtistBySlug retrieves an artist by slug
func (r *Repository) GetArtistBySlug(slug string) (Artist, bool) {
	id, exists := r.artistSlugs[slug]
	if !exists {
		return Artist{}, false
	}
	return r.GetArtist(id)
}

// GetAllRelations returns all relations
func (r *Repository) GetAllRelations() []Relation {
	relations := make([]Relation, 0, len(r.relations))
	for _, relation := range r.relations {
		relations = append(relations, relation)
	}
	return relations
}

// GetRelation retrieves a relation by ID
func (r *Repository) GetRelation(id int) (Relation, bool) {
	relation, exists := r.relations[id]
	return relation, exists
}

// GetUniqueLocations returns cached unique locations
func (r *Repository) GetUniqueLocations() []string {
	return r.uniqueLocations
}

// GetUniqueDates returns cached unique dates
func (r *Repository) GetUniqueDates() []string {
	return r.uniqueDates
}

// GetStats returns precomputed statistics
func (r *Repository) GetStats() map[string]int {
	return r.stats
}

// GetTotalMembers returns the total number of band members across all artists
func (r *Repository) GetTotalMembers() int {
	total := 0
	for _, artist := range r.artists {
		total += len(artist.Members)
	}
	return total
}

// GetTotalCountries returns the number of unique countries present in locations
func (r *Repository) GetTotalCountries() int {
	countrySet := make(map[string]bool)
	for _, loc := range r.uniqueLocations {
		parts := strings.Split(loc, "-")
		if len(parts) >= 2 {
			country := strings.TrimSpace(parts[len(parts)-1])
			countrySet[country] = true
		}
	}
	return len(countrySet)
}

// -----------------------------
// Business Logic Methods
// -----------------------------

// GetAllArtistsSorted returns artists sorted by name
func (r *Repository) GetAllArtistsSorted() []Artist {
	artists := r.GetAllArtists()
	sort.Slice(artists, func(i, j int) bool {
		return artists[i].Name < artists[j].Name
	})
	return artists
}

// GetArtistNavigation returns previous and next artists for navigation
func (r *Repository) GetArtistNavigation(currentArtist Artist) (prevArtist *Artist, nextArtist *Artist) {
	artists := r.GetAllArtistsSorted()

	for i, artist := range artists {
		if artist.ID == currentArtist.ID {
			if i > 0 {
				prevArtist = &artists[i-1]
			}
			if i < len(artists)-1 {
				nextArtist = &artists[i+1]
			}
			break
		}
	}

	return prevArtist, nextArtist
}

// CalculateLocationStats returns precomputed location statistics
func (r *Repository) CalculateLocationStats() []LocationStat {
	return r.locationStats
}

// GetLocationDetailsBySlug retrieves detailed location information by slug
func (r *Repository) GetLocationDetailsBySlug(slug string) (LocationDetail, bool) {
	// Find location by slug
	var targetLocation string
	for _, location := range r.uniqueLocations {
		if GenerateLocationSlug(location) == slug {
			targetLocation = location
			break
		}
	}

	if targetLocation == "" {
		return LocationDetail{}, false
	}

	// Find artists and dates for this location
	var artists []Artist
	var dates []string
	artistsWithDates := make([]ArtistWithDates, 0)
	dateSet := make(map[string]bool)

	for _, relation := range r.relations {
		if concertDates, exists := relation.DatesLocations[targetLocation]; exists {
			if artist, exists := r.artists[relation.ID]; exists {
				artists = append(artists, artist)
				artistsWithDates = append(artistsWithDates, ArtistWithDates{
					Artist: artist,
					Dates:  concertDates,
				})

				for _, date := range concertDates {
					if !dateSet[date] {
						dates = append(dates, date)
						dateSet[date] = true
					}
				}
			}
		}
	}

	// Sort dates
	sort.Strings(dates)

	return LocationDetail{
		Name:             targetLocation,
		DisplayName:      NormalizeLocationName(targetLocation),
		Slug:             GenerateLocationSlug(targetLocation),
		Artists:          artists,
		ArtistsWithDates: artistsWithDates,
		Dates:            dates,
		ArtistCount:      len(artists),
		ConcertCount:     len(dates),
	}, true
}

// GetArtistsWithDatesForLocation returns artists with their dates for a specific location
func (r *Repository) GetArtistsWithDatesForLocation(locationName string) []ArtistWithDates {
	var result []ArtistWithDates

	for _, relation := range r.relations {
		if dates, exists := relation.DatesLocations[locationName]; exists {
			if artist, exists := r.artists[relation.ID]; exists {
				result = append(result, ArtistWithDates{
					Artist: artist,
					Dates:  dates,
				})
			}
		}
	}

	return result
}

// CalculateTotalShows calculates total shows for a relation
func (r *Repository) CalculateTotalShows(relation Relation) int {
	total := 0
	for _, dates := range relation.DatesLocations {
		total += len(dates)
	}
	return total
}

// ExtractCountries extracts unique countries from a relation
func (r *Repository) ExtractCountries(relation Relation) []string {
	countrySet := make(map[string]bool)

	for location := range relation.DatesLocations {
		parts := strings.Split(location, "-")
		if len(parts) >= 2 {
			country := strings.ToUpper(strings.TrimSpace(parts[len(parts)-1]))
			countrySet[country] = true
		}
	}

	countries := make([]string, 0, len(countrySet))
	for country := range countrySet {
		countries = append(countries, country)
	}

	sort.Strings(countries)
	return countries
}

// -----------------------------
// Precomputation Methods
// -----------------------------

// precomputeIndexes calculates and caches all derived data for performance
func (r *Repository) precomputeIndexes() {
	r.computeUniqueLocationsAndDates()
	r.computeLocationStats()
	r.computeGlobalStats()
}

// computeUniqueLocationsAndDates extracts unique locations and dates
func (r *Repository) computeUniqueLocationsAndDates() {
	locationSet := make(map[string]bool)
	dateSet := make(map[string]bool)

	for _, relation := range r.relations {
		for location, dates := range relation.DatesLocations {
			locationSet[location] = true
			for _, date := range dates {
				dateSet[date] = true
			}
		}
	}

	// Convert to sorted slices
	r.uniqueLocations = make([]string, 0, len(locationSet))
	for location := range locationSet {
		r.uniqueLocations = append(r.uniqueLocations, location)
	}
	sort.Strings(r.uniqueLocations)

	r.uniqueDates = make([]string, 0, len(dateSet))
	for date := range dateSet {
		r.uniqueDates = append(r.uniqueDates, date)
	}
	sort.Strings(r.uniqueDates)
}

// computeLocationStats precomputes statistics for all locations
func (r *Repository) computeLocationStats() {
	locationData := make(map[string]*LocationStat)

	// Initialize location stats
	for _, location := range r.uniqueLocations {
		locationData[location] = &LocationStat{
			Name:        location,
			DisplayName: NormalizeLocationName(location),
			Slug:        GenerateLocationSlug(location),
			Artists:     make([]Artist, 0),
			Dates:       make([]string, 0),
		}
	}

	// Populate with relation data
	for _, relation := range r.relations {
		artist, exists := r.artists[relation.ID]
		if !exists {
			continue
		}

		for location, dates := range relation.DatesLocations {
			if stat, exists := locationData[location]; exists {
				stat.Artists = append(stat.Artists, artist)
				stat.Dates = append(stat.Dates, dates...)
				stat.ArtistCount++
				stat.ConcertCount += len(dates)
			}
		}
	}

	// Convert to slice and sort by artist count (descending)
	r.locationStats = make([]LocationStat, 0, len(locationData))
	for _, stat := range locationData {
		r.locationStats = append(r.locationStats, *stat)
	}

	sort.Slice(r.locationStats, func(i, j int) bool {
		return r.locationStats[i].ArtistCount > r.locationStats[j].ArtistCount
	})
}

// computeGlobalStats precomputes global statistics
func (r *Repository) computeGlobalStats() {
	totalConcerts := 0
	for _, relation := range r.relations {
		for _, dates := range relation.DatesLocations {
			totalConcerts += len(dates)
		}
	}

	r.stats = map[string]int{
		"artists":        len(r.artists),
		"locations":      len(r.uniqueLocations),
		"dates":          len(r.uniqueDates),
		"relations":      len(r.relations),
		"total_concerts": totalConcerts,
	}
}

// -----------------------------
// Utility Functions
// -----------------------------

// generateArtistSlug creates a URL-friendly slug from artist name
func generateArtistSlug(name string) string {
	if name == "" {
		return ""
	}

	// Convert to lowercase
	slug := strings.ToLower(name)

	// Replace spaces and special characters with hyphens
	reg := regexp.MustCompile(`[^a-z0-9]+`)
	slug = reg.ReplaceAllString(slug, "-")

	// Remove leading and trailing hyphens
	slug = strings.Trim(slug, "-")

	// Replace multiple consecutive hyphens with single hyphen
	reg = regexp.MustCompile(`-+`)
	slug = reg.ReplaceAllString(slug, "-")

	return slug
}

// GenerateLocationSlug creates a URL-friendly slug from a location name
func GenerateLocationSlug(locationName string) string {
	if locationName == "" {
		return ""
	}

	// Convert to lowercase
	slug := strings.ToLower(locationName)

	// Replace underscores with hyphens for consistency
	slug = strings.ReplaceAll(slug, "_", "-")

	// Keep only alphanumeric characters and hyphens
	reg := regexp.MustCompile(`[^a-z0-9-]+`)
	slug = reg.ReplaceAllString(slug, "-")

	// Remove leading and trailing hyphens
	slug = strings.Trim(slug, "-")

	// Replace multiple consecutive hyphens with single hyphen
	reg = regexp.MustCompile(`-+`)
	slug = reg.ReplaceAllString(slug, "-")

	return slug
}

// NormalizeLocationName formats location names for display
func NormalizeLocationName(locationName string) string {
	if locationName == "" {
		return ""
	}

	// Split by the last hyphen to separate city from country
	parts := strings.Split(locationName, "-")
	if len(parts) < 2 {
		return toTitleCase(strings.ReplaceAll(locationName, "_", " "))
	}

	// Get city and country parts
	countryIndex := len(parts) - 1
	country := parts[countryIndex]
	city := strings.Join(parts[:countryIndex], "-")

	// Format city: replace underscores with spaces and title case
	formattedCity := toTitleCase(strings.ReplaceAll(city, "_", " "))

	// Format country: uppercase
	formattedCountry := strings.ToUpper(country)

	return fmt.Sprintf("%s, %s", formattedCity, formattedCountry)
}

// toTitleCase converts a string to title case
func toTitleCase(s string) string {
	words := strings.Fields(s)
	for i, word := range words {
		if len(word) > 0 {
			words[i] = strings.ToUpper(string(word[0])) + strings.ToLower(word[1:])
		}
	}
	return strings.Join(words, " ")
}
