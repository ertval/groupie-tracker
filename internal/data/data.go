// Package data provides application domain models and repository for the Groupie Tracker application.
package data

import (
	"context"
	"fmt"
	"log"
	"regexp"
	"sort"
	"strings"

	"groupie-tracker/internal/client"
)

// Artist represents a musical artist in the application domain.
type Artist struct {
	ID           int
	Name         string
	Members      []string
	CreationYear int
	FirstAlbum   string
	Image        string
	Slug         string // SEO-friendly URL slug
}

// Relation represents concert relationships in the application domain.
type Relation struct {
	ID             int
	DatesLocations map[string][]string
}

// LocationStat represents statistics for a location.
type LocationStat struct {
	Name         string
	DisplayName  string
	Slug         string
	ArtistCount  int
	ConcertCount int
	Artists      []Artist
	Dates        []string
}

// ArtistWithDates pairs an artist with their concert dates at a location.
type ArtistWithDates struct {
	Artist Artist
	Dates  []string
}

// LocationDetail provides detailed information about a location.
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

// Repository provides unified data storage and retrieval for the application.
type Repository struct {
	artists         map[int]Artist
	relations       map[int]Relation
	artistSlugs     map[string]int // slug -> artist ID
	uniqueLocations []string
	uniqueDates     []string
	locationStats   []LocationStat
	stats           map[string]int
}

// APIClient defines the subset of the external client used by the repository.
// Using an interface improves testability (allows mocks in tests).
type APIClient interface {
	FetchAll(ctx context.Context) (*client.Response, error)
}

// NewRepository creates a new empty repository.
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

// InitializeWithAPIClient fetches data from the API and initializes the repository.
// This is the recommended method for standard initialization.
func (r *Repository) InitializeWithAPIClient(ctx context.Context, apiClient APIClient) error {
	if apiClient == nil {
		return fmt.Errorf("API client cannot be nil")
	}

	apiData, err := apiClient.FetchAll(ctx)
	if err != nil {
		return fmt.Errorf("failed to fetch data from API: %w", err)
	}

	if apiData == nil {
		return fmt.Errorf("API returned nil data")
	}

	if len(apiData.Artists) == 0 {
		return fmt.Errorf("no artists data received from API")
	}

	r.loadData(apiData)
	return nil
}

// Initialize loads and validates repository data. The caller may provide either
// - an already-fetched `apiData` (preferred when caller manages fetching), or
// - an `apiClient` to fetch data inside this method.
// If both are provided, `apiData` takes precedence. At least one must be non-nil.
func (r *Repository) Initialize(ctx context.Context, apiClient APIClient, apiData *client.Response) error {
	// Choose data source: prefer explicit apiData when provided.
	var dataToLoad *client.Response

	if apiData != nil {
		dataToLoad = apiData
	} else if apiClient != nil {
		fetched, err := apiClient.FetchAll(ctx)
		if err != nil {
			return fmt.Errorf("failed to fetch data from API: %w", err)
		}
		dataToLoad = fetched
	} else {
		return fmt.Errorf("either apiClient or apiData must be provided")
	}

	if dataToLoad == nil {
		return fmt.Errorf("API data cannot be nil")
	}

	if len(dataToLoad.Artists) == 0 {
		return fmt.Errorf("no artists data received from API")
	}

	r.loadData(dataToLoad)
	return nil
}

func (r *Repository) loadData(apiData *client.Response) {
	// Clear existing data
	r.artists = make(map[int]Artist)
	r.relations = make(map[int]Relation)
	r.artistSlugs = make(map[string]int)

	// Convert API artists to domain artists
	for _, apiArtist := range apiData.Artists {
		artist := Artist{
			ID:           apiArtist.ID,
			Name:         apiArtist.Name,
			Members:      apiArtist.Members,
			CreationYear: apiArtist.CreationYear,
			FirstAlbum:   apiArtist.FirstAlbum,
			Image:        apiArtist.Image,
			Slug:         generateSlug(apiArtist.Name),
		}
		r.artists[artist.ID] = artist
		r.artistSlugs[artist.Slug] = artist.ID
	}

	// Convert API relations to domain relations
	for _, apiRelation := range apiData.Relations {
		r.relations[apiRelation.ID] = Relation{
			ID:             apiRelation.ID,
			DatesLocations: apiRelation.DatesLocations,
		}
	}

	// Precompute derived data
	r.precompute()

	log.Printf("✅ Repository loaded: %d artists, %d relations",
		len(r.artists), len(r.relations))
}

// Core Data Access Methods
// -----------------------------

// GetAllArtists returns all artists.
func (r *Repository) GetAllArtists() []Artist {
	artists := make([]Artist, 0, len(r.artists))
	for _, artist := range r.artists {
		artists = append(artists, artist)
	}
	return artists
}

// GetAllArtistsSorted returns artists sorted by name.
func (r *Repository) GetAllArtistsSorted() []Artist {
	artists := r.GetAllArtists()
	sort.Slice(artists, func(i, j int) bool {
		return artists[i].Name < artists[j].Name
	})
	return artists
}

// GetArtist retrieves an artist by ID.
func (r *Repository) GetArtist(id int) (Artist, bool) {
	artist, exists := r.artists[id]
	return artist, exists
}

// GetArtistBySlug retrieves an artist by slug.
func (r *Repository) GetArtistBySlug(slug string) (Artist, bool) {
	id, exists := r.artistSlugs[slug]
	if !exists {
		return Artist{}, false
	}
	return r.GetArtist(id)
}

// GetRelation retrieves a relation by ID.
func (r *Repository) GetRelation(id int) (Relation, bool) {
	relation, exists := r.relations[id]
	return relation, exists
}

// GetUniqueLocations returns cached unique locations.
func (r *Repository) GetUniqueLocations() []string {
	return r.uniqueLocations
}

// GetStats returns precomputed statistics.
func (r *Repository) GetStats() map[string]int {
	return r.stats
}

// GetTotalMembers returns the total number of band members across all artists.
func (r *Repository) GetTotalMembers() int {
	total := 0
	for _, artist := range r.artists {
		total += len(artist.Members)
	}
	return total
}

// GetTotalCountries returns the number of unique countries present in locations.
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

// GetArtistNavigation returns previous and next artists for navigation.
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

// CalculateLocationStats returns precomputed location statistics.
func (r *Repository) CalculateLocationStats() []LocationStat {
	return r.locationStats
}

// GetLocationDetailsBySlug retrieves detailed location information by slug.
func (r *Repository) GetLocationDetailsBySlug(slug string) (LocationDetail, bool) {
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

// GetArtistsWithDatesForLocation returns artists with their dates for a specific location.
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

// CalculateTotalShows calculates total shows for a relation.
func (r *Repository) CalculateTotalShows(relation Relation) int {
	total := 0
	for _, dates := range relation.DatesLocations {
		total += len(dates)
	}
	return total
}

// ExtractCountries extracts unique countries from a relation.
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

// precompute calculates and caches all derived data for performance.
func (r *Repository) precompute() {
	r.computeUniqueLocationsAndDates()
	r.computeLocationStats()
	r.computeGlobalStats()
}

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

func (r *Repository) computeLocationStats() {
	locationData := make(map[string]*LocationStat)

	for _, location := range r.uniqueLocations {
		locationData[location] = &LocationStat{
			Name:        location,
			DisplayName: NormalizeLocationName(location),
			Slug:        GenerateLocationSlug(location),
			Artists:     make([]Artist, 0),
			Dates:       make([]string, 0),
		}
	}

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

	r.locationStats = make([]LocationStat, 0, len(locationData))
	for _, stat := range locationData {
		r.locationStats = append(r.locationStats, *stat)
	}

	sort.Slice(r.locationStats, func(i, j int) bool {
		return r.locationStats[i].ArtistCount > r.locationStats[j].ArtistCount
	})
}

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

// generateSlug creates a URL-friendly slug from artist name.
func generateSlug(name string) string {
	if name == "" {
		return ""
	}

	slug := strings.ToLower(name)
	reg := regexp.MustCompile(`[^a-z0-9]+`)
	slug = reg.ReplaceAllString(slug, "-")
	slug = strings.Trim(slug, "-")
	reg = regexp.MustCompile(`-+`)
	slug = reg.ReplaceAllString(slug, "-")
	return slug
}

// GenerateLocationSlug creates a URL-friendly slug from a location name.
func GenerateLocationSlug(locationName string) string {
	if locationName == "" {
		return ""
	}

	slug := strings.ToLower(locationName)
	slug = strings.ReplaceAll(slug, "_", "-")
	reg := regexp.MustCompile(`[^a-z0-9-]+`)
	slug = reg.ReplaceAllString(slug, "-")
	slug = strings.Trim(slug, "-")
	reg = regexp.MustCompile(`-+`)
	slug = reg.ReplaceAllString(slug, "-")
	return slug
}

// NormalizeLocationName formats location names for display.
func NormalizeLocationName(locationName string) string {
	if locationName == "" {
		return ""
	}

	parts := strings.Split(locationName, "-")
	if len(parts) < 2 {
		return toTitleCase(strings.ReplaceAll(locationName, "_", " "))
	}

	countryIndex := len(parts) - 1
	country := parts[countryIndex]
	city := strings.Join(parts[:countryIndex], "-")

	formattedCity := toTitleCase(strings.ReplaceAll(city, "_", " "))
	formattedCountry := strings.ToUpper(country)

	return fmt.Sprintf("%s, %s", formattedCity, formattedCountry)
}

func toTitleCase(s string) string {
	words := strings.Fields(s)
	for i, word := range words {
		if len(word) > 0 {
			words[i] = strings.ToUpper(string(word[0])) + strings.ToLower(word[1:])
		}
	}
	return strings.Join(words, " ")
}
