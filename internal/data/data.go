// Package data provides unified data structures, repository, and business logic for the Groupie Tracker application.
//
// This package follows SOLID principles and provides a simple, unified data access layer.
// S: Single Responsibility Principle — Each module/class should have one responsibility.
// O: Open/Closed Principle — Software entities should be open for extension, but closed for modification.
// L: Liskov Substitution Principle — Subtypes must be substitutable for their base types.
// I: Interface Segregation Principle — Prefer several specific interfaces over one general-purpose interface.
// D: Dependency Inversion Principle — Depend on abstractions, not concrete implementations.

package data

import (
	"context"
	"errors"
	"fmt"
	"log"
	"regexp"
	"sort"
	"strings"
	"time"
)

// -----------------------------
// Structs (Core Data Models)
// -----------------------------

// (API Struct) Artist represents a musical artist or band with all their information from the API.
type Artist struct {
	ID           int      `json:"id"`
	Image        string   `json:"image"`
	Name         string   `json:"name"`
	Members      []string `json:"members"`
	CreationYear int      `json:"creationDate"`
	FirstAlbum   string   `json:"firstAlbum"`
	Slug         string   `json:"slug,omitempty"`
}

// (API Struct) Relation represents the relationship between artists with their concert locations and dates from the API.
type Relation struct {
	ID             int                 `json:"id"`
	DatesLocations map[string][]string `json:"datesLocations"`
}

// (API Struct) Location represents a location data structure from the API
type Location struct {
	ID        int      `json:"id"`
	Locations []string `json:"locations"`
}

// (API Struct) Date represents a date data structure from the API
type Date struct {
	ID    int      `json:"id"`
	Dates []string `json:"dates"`
}

// (Custom Data Struct) LocationStat represents statistics for a location.
type LocationStat struct {
	Name         string
	DisplayName  string
	Slug         string
	ArtistCount  int
	ConcertCount int
	Artists      []Artist
	Dates        []string
}

// (Custom Data Struct) ArtistWithDates pairs an artist with the concert dates they played at a location.
type ArtistWithDates struct {
	Artist Artist
	Dates  []string
}

// (Custom Data Struct) PageData represents common data needed for all pages.
type PageData struct {
	Title    string
	ExtraCSS string
	ExtraJS  string
}

// API and Response Types
// -----------------------------

// APIResponse represents the main API response structure.
type APIResponse struct {
	Artists   []Artist   `json:"artists,omitempty"`
	Locations []Location `json:"locations,omitempty"`
	Dates     []Date     `json:"dates,omitempty"`
	Relations []Relation `json:"relations,omitempty"`
}

// APIClient defines the interface for fetching data from external APIs.
type APIClient interface {
	FetchAllData(ctx context.Context) (*APIResponse, error)
}

// -----------------------------
// Repository (Initialization & Validation)
// -----------------------------

// Repository provides a simple data store and business logic for all Groupie Tracker data.
type Repository struct {
	// Core data maps
	artists     map[int]Artist
	artistSlugs map[string]int // slug -> artist ID mapping
	relations   map[int]Relation

	// Pre-computed data for performance
	uniqueLocations []string
	uniqueDates     []string
}

// NewRepository creates a new empty repository.
func NewRepository() *Repository {
	return &Repository{
		artists:         make(map[int]Artist),
		artistSlugs:     make(map[string]int),
		relations:       make(map[int]Relation),
		uniqueLocations: make([]string, 0),
		uniqueDates:     make([]string, 0),
	}
}

// InitializeWithAPI loads data from the API client into the repository.
// This should be called once at application startup.
func (r *Repository) InitializeWithAPI(ctx context.Context, api APIClient) error {
	data, err := api.FetchAllData(ctx)
	if err != nil {
		return fmt.Errorf("failed to fetch data from API: %w", err)
	}

	r.LoadData(*data)
	return nil
}

// LoadData loads all data from an APIResponse into the repository.
func (r *Repository) LoadData(data APIResponse) {
	// Clear existing data
	r.artists = make(map[int]Artist)
	r.artistSlugs = make(map[string]int)
	r.relations = make(map[int]Relation)

	// Load artists and generate slugs
	for _, artist := range data.Artists {
		artist.SetSlug() // Generate slug for SEO-friendly URLs
		r.artists[artist.ID] = artist
		r.artistSlugs[artist.GetSlug()] = artist.ID
	}

	// Load relations
	for _, relation := range data.Relations {
		r.relations[relation.ID] = relation
	}

	// Pre-compute unique locations and dates for performance
	r.computeUniqueData()

	log.Printf("✅ Repository loaded: %d artists, %d relations",
		len(r.artists), len(r.relations))
}

// Validate checks if the Relation struct has valid data.
func (r *Relation) Validate() error {
	if r.ID <= 0 {
		return errors.New("relation ID must be greater than 0")
	}

	if len(r.DatesLocations) == 0 {
		return errors.New("relation must have at least one dates-location mapping")
	}

	return nil
}

// Validate checks if the Artist struct has valid data.
func (a *Artist) Validate() error {
	if a.Name == "" {
		return errors.New("artist name cannot be empty")
	}

	if a.CreationYear <= 0 {
		return errors.New("creation year must be greater than 0")
	}

	if len(a.Members) == 0 {
		return errors.New("artist must have at least one member")
	}

	return nil
}

// GetFirstAlbumDate parses the FirstAlbum string and returns a time.Time.
// Expected format is "DD-MM-YYYY".
func (a *Artist) GetFirstAlbumDate() (time.Time, error) {
	if a.FirstAlbum == "" {
		return time.Time{}, errors.New("first album date is empty")
	}

	// Parse the date in DD-MM-YYYY format
	parsedTime, err := time.Parse("02-01-2006", a.FirstAlbum)
	if err != nil {
		return time.Time{}, errors.New("invalid date format, expected DD-MM-YYYY")
	}

	return parsedTime, nil
}

// GenerateSlug creates a URL-friendly slug from the artist name.
func (a *Artist) GenerateSlug() string {
	if a.Name == "" {
		return ""
	}

	// Convert to lowercase
	slug := strings.ToLower(a.Name)

	// Replace spaces and special characters with hyphens
	// Keep only alphanumeric characters and hyphens
	reg := regexp.MustCompile(`[^a-z0-9]+`)
	slug = reg.ReplaceAllString(slug, "-")

	// Remove leading and trailing hyphens
	slug = strings.Trim(slug, "-")

	// Replace multiple consecutive hyphens with single hyphen
	reg = regexp.MustCompile(`-+`)
	slug = reg.ReplaceAllString(slug, "-")

	return slug
}

// SetSlug generates and sets the slug for the artist.
func (a *Artist) SetSlug() {
	a.Slug = a.GenerateSlug()
}

// GetSlug returns the artist's slug, generating it if it doesn't exist.
func (a *Artist) GetSlug() string {
	if a.Slug == "" {
		a.Slug = a.GenerateSlug()
	}
	return a.Slug
}

// -----------------------------
// Basic Getters / Setters
// -----------------------------

// GetAllArtists returns all artists (raw data, no sorting).
func (r *Repository) GetAllArtists() []Artist {
	artists := make([]Artist, 0, len(r.artists))
	for _, artist := range r.artists {
		artists = append(artists, artist)
	}
	return artists
}

// GetArtist retrieves an artist by ID.
func (r *Repository) GetArtist(id int) (Artist, bool) {
	artist, exists := r.artists[id]
	return artist, exists
}

// GetArtistBySlug retrieves an artist by their slug.
func (r *Repository) GetArtistBySlug(slug string) (Artist, bool) {
	artistID, exists := r.artistSlugs[slug]
	if !exists {
		return Artist{}, false
	}

	artist, exists := r.artists[artistID]
	return artist, exists
}

// GetAllRelations returns all relations.
func (r *Repository) GetAllRelations() []Relation {
	relations := make([]Relation, 0, len(r.relations))
	for _, relation := range r.relations {
		relations = append(relations, relation)
	}
	return relations
}

// GetRelation retrieves a relation by ID.
func (r *Repository) GetRelation(id int) (Relation, bool) {
	relation, exists := r.relations[id]
	return relation, exists
}

// GetUniqueLocations returns a slice of unique location strings (raw data, no sorting).
func (r *Repository) GetUniqueLocations() []string {
	// Return a copy to prevent external modification
	result := make([]string, len(r.uniqueLocations))
	copy(result, r.uniqueLocations)
	return result
}

// GetUniqueDates returns a slice of unique date strings (raw data, no sorting).
func (r *Repository) GetUniqueDates() []string {
	// Return a copy to prevent external modification
	result := make([]string, len(r.uniqueDates))
	copy(result, r.uniqueDates)
	return result
}

// -----------------------------
// Calculation Functions (business logic)
// -----------------------------

// GetAllArtistsSorted returns all artists sorted alphabetically by name.
func (r *Repository) GetAllArtistsSorted() []Artist {
	artists := r.GetAllArtists()
	sort.Slice(artists, func(i, j int) bool {
		return strings.ToLower(artists[i].Name) < strings.ToLower(artists[j].Name)
	})
	return artists
}

// GetArtistNavigation returns the previous and next artists for navigation based on alphabetical ordering.
func (r *Repository) GetArtistNavigation(currentArtist Artist) (prevArtist *Artist, nextArtist *Artist) {
	allArtists := r.GetAllArtistsSorted()
	currentIndex := -1

	// Find current artist index
	for i, a := range allArtists {
		if a.ID == currentArtist.ID {
			currentIndex = i
			break
		}
	}

	if currentIndex == -1 {
		return nil, nil // Artist not found in list
	}

	// Get previous artist
	if currentIndex > 0 {
		prev := allArtists[currentIndex-1]
		prevArtist = &prev
	}

	// Get next artist
	if currentIndex < len(allArtists)-1 {
		next := allArtists[currentIndex+1]
		nextArtist = &next
	}

	return prevArtist, nextArtist
}

// CalculateLocationStats calculates statistics for each location.
func (r *Repository) CalculateLocationStats() []LocationStat {
	locationMap := make(map[string]*LocationStat)
	allArtists := r.GetAllArtists()
	allRelations := r.GetAllRelations()

	// Create a map of artist ID to artist for quick lookup
	artistMap := make(map[int]Artist)
	for _, artist := range allArtists {
		artistMap[artist.ID] = artist
	}

	// Process each relation to build location statistics
	for _, relation := range allRelations {
		artist, exists := artistMap[relation.ID]
		if !exists {
			continue
		}

		for location, dates := range relation.DatesLocations {
			if locationMap[location] == nil {
				locationMap[location] = &LocationStat{
					Name:        location,
					DisplayName: NormalizeLocationName(location),
					Slug:        GenerateLocationSlug(location),
					Artists:     []Artist{},
				}
			}

			locationMap[location].ArtistCount++
			locationMap[location].ConcertCount += len(dates)
			locationMap[location].Artists = append(locationMap[location].Artists, artist)
			locationMap[location].Dates = append(locationMap[location].Dates, dates...)
		}
	}

	// Convert map to slice and sort by concert count (descending)
	var locationStats []LocationStat
	for _, stat := range locationMap {
		// Remove duplicate dates and sort them
		dateSet := make(map[string]bool)
		for _, date := range stat.Dates {
			dateSet[date] = true
		}
		stat.Dates = make([]string, 0, len(dateSet))
		for date := range dateSet {
			stat.Dates = append(stat.Dates, date)
		}
		sort.Strings(stat.Dates)

		locationStats = append(locationStats, *stat)
	}

	// Sort by concert count in descending order (most popular first)
	sort.Slice(locationStats, func(i, j int) bool {
		return locationStats[i].ConcertCount > locationStats[j].ConcertCount
	})

	return locationStats
}

// GetLocationDetailsBySlug returns detailed information about a specific location
func (r *Repository) GetLocationDetailsBySlug(locationSlug string) (*LocationStat, bool) {
	locationStats := r.CalculateLocationStats()
	for _, stat := range locationStats {
		if stat.Slug == locationSlug {
			return &stat, true
		}
	}
	return nil, false
}

// GetArtistsWithDatesForLocation returns a slice of ArtistWithDates for a specific location.
func (r *Repository) GetArtistsWithDatesForLocation(locationName string) []ArtistWithDates {
	allRelations := r.GetAllRelations()
	artistDates := make(map[int]map[string]bool)
	artistOrder := []int{}

	for _, rel := range allRelations {
		if dates, ok := rel.DatesLocations[locationName]; ok {
			if _, exists := artistDates[rel.ID]; !exists {
				artistDates[rel.ID] = make(map[string]bool)
				artistOrder = append(artistOrder, rel.ID)
			}
			for _, d := range dates {
				artistDates[rel.ID][d] = true
			}
		}
	}

	// Build result
	var result []ArtistWithDates
	artistMap := make(map[int]Artist)
	for _, a := range r.GetAllArtists() {
		artistMap[a.ID] = a
	}

	for _, id := range artistOrder {
		datesMap := artistDates[id]
		if datesMap == nil {
			continue
		}
		dates := make([]string, 0, len(datesMap))
		for d := range datesMap {
			dates = append(dates, d)
		}
		sort.Strings(dates)

		if artist, ok := artistMap[id]; ok {
			result = append(result, ArtistWithDates{Artist: artist, Dates: dates})
		}
	}

	return result
}

// CalculateTotalShows calculates the total number of shows for an artist
func (r *Repository) CalculateTotalShows(relation Relation) int {
	total := 0
	for _, dates := range relation.DatesLocations {
		total += len(dates)
	}
	return total
}

// ExtractCountries extracts unique countries from relation data
func (r *Repository) ExtractCountries(relation Relation) []string {
	countrySet := make(map[string]bool)

	for location := range relation.DatesLocations {
		// Extract country from location (assuming format "city-country")
		parts := strings.Split(location, "-")
		if len(parts) >= 2 {
			country := strings.TrimSpace(parts[len(parts)-1])
			countrySet[country] = true
		}
	}

	countries := make([]string, 0, len(countrySet))
	for country := range countrySet {
		countries = append(countries, country)
	}

	return countries
}

// GetStats returns comprehensive statistics about the stored data.
func (r *Repository) GetStats() map[string]int {
	allArtists := r.GetAllArtists()
	allRelations := r.GetAllRelations()
	locations := r.GetUniqueLocations()
	dates := r.GetUniqueDates()

	// Calculate total concerts
	totalConcerts := 0
	for _, relation := range allRelations {
		for _, concertDates := range relation.DatesLocations {
			totalConcerts += len(concertDates)
		}
	}

	return map[string]int{
		"artists":        len(allArtists),
		"locations":      len(locations),
		"dates":          len(dates),
		"relations":      len(allRelations),
		"total_concerts": totalConcerts,
	}
}

// -----------------------------
// Exported Helpers
// -----------------------------

// GenerateLocationSlug creates a URL-friendly slug from a location name.
// Location names come in formats like "new_york-usa" or "paris-france"
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
// Converts "new_york-usa" to "New York, USA"
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

// -----------------------------
// Internal Utilities
// -----------------------------

// computeUniqueData pre-computes unique locations and dates for performance.
func (r *Repository) computeUniqueData() {
	locationSet := make(map[string]bool)
	dateSet := make(map[string]bool)

	// Extract unique locations from relations
	for _, relation := range r.relations {
		for location, dates := range relation.DatesLocations {
			locationSet[location] = true
			for _, date := range dates {
				dateSet[date] = true
			}
		}
	}

	// Convert sets to slices (no sorting - let service layer handle that)
	r.uniqueLocations = make([]string, 0, len(locationSet))
	for location := range locationSet {
		r.uniqueLocations = append(r.uniqueLocations, location)
	}

	r.uniqueDates = make([]string, 0, len(dateSet))
	for date := range dateSet {
		r.uniqueDates = append(r.uniqueDates, date)
	}
}

// toTitleCase converts a string to title case (first letter of each word capitalized)
func toTitleCase(s string) string {
	words := strings.Fields(s)
	for i, word := range words {
		if len(word) > 0 {
			words[i] = strings.ToUpper(string(word[0])) + strings.ToLower(word[1:])
		}
	}
	return strings.Join(words, " ")
}
