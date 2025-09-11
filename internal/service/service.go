// Package service provides business logic for Groupie Tracker data.
package service

import (
	"sort"
	"strings"

	"groupie-tracker/internal/models"
)

// DataStore defines the interface for reading data from the store.
type DataStore interface {
	GetAllArtists() []models.Artist
	GetAllRelations() []models.Relation
	GetUniqueLocations() []string
	GetUniqueDates() []string
}

// Service provides business logic operations for Groupie Tracker data.
type Service struct {
	store DataStore
}

// LocationStat represents statistics for a location.
type LocationStat struct {
	Name         string
	ArtistCount  int
	ConcertCount int
	Artists      []models.Artist
}

// LocationFrequency represents the frequency of concerts at a location.
type LocationFrequency struct {
	Location string
	Count    int
}

// ArtistWithDates pairs an artist with the concert dates they played at a location.
type ArtistWithDates struct {
	Artist models.Artist
	Dates  []string
}

// NewService creates a new service instance.
func NewService(store DataStore) *Service {
	return &Service{
		store: store,
	}
}

// CalculateLocationStats calculates statistics for each location.
func (s *Service) CalculateLocationStats() []LocationStat {
	locationMap := make(map[string]*LocationStat)
	allArtists := s.store.GetAllArtists()
	allRelations := s.store.GetAllRelations()

	// Create a map of artist ID to artist for quick lookup
	artistMap := make(map[int]models.Artist)
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
					Name:         location,
					ArtistCount:  0,
					ConcertCount: 0,
					Artists:      []models.Artist{},
				}
			}

			locationMap[location].ArtistCount++
			locationMap[location].ConcertCount += len(dates)
			locationMap[location].Artists = append(locationMap[location].Artists, artist)
		}
	}

	// Convert map to slice and sort by concert count (descending)
	var locationStats []LocationStat
	for _, stat := range locationMap {
		locationStats = append(locationStats, *stat)
	}

	// Sort by concert count in descending order (most popular first)
	sort.Slice(locationStats, func(i, j int) bool {
		return locationStats[i].ConcertCount > locationStats[j].ConcertCount
	})

	return locationStats
}

// SortLocationStatsByConcertCount sorts location statistics by concert count in descending order.
func (s *Service) SortLocationStatsByConcertCount(stats []LocationStat) []LocationStat {
	// Create a copy to avoid modifying the original slice
	sorted := make([]LocationStat, len(stats))
	copy(sorted, stats)

	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].ConcertCount > sorted[j].ConcertCount
	})

	return sorted
}

// CalculateTotalCountries calculates the total number of unique countries.
func (s *Service) CalculateTotalCountries(locationStats []LocationStat) int {
	countrySet := make(map[string]bool)

	for _, stat := range locationStats {
		// Extract country from location (assuming format "city-country")
		parts := strings.Split(stat.Name, "-")
		if len(parts) >= 2 {
			country := strings.TrimSpace(parts[len(parts)-1])
			countrySet[country] = true
		}
	}

	return len(countrySet)
}

// CalculateTotalConcerts calculates the total number of concerts across all artists.
func (s *Service) CalculateTotalConcerts() int {
	total := 0
	allRelations := s.store.GetAllRelations()

	for _, relation := range allRelations {
		for _, dates := range relation.DatesLocations {
			total += len(dates)
		}
	}

	return total
}

// GetMostPopularLocations returns the most frequently used concert locations.
// Returns up to 'limit' locations sorted by frequency (most frequent first).
func (s *Service) GetMostPopularLocations(limit int) []LocationFrequency {
	allRelations := s.store.GetAllRelations()
	locationCount := make(map[string]int)

	// Count occurrences of each location
	for _, relation := range allRelations {
		for location, dates := range relation.DatesLocations {
			locationCount[location] += len(dates) // Count concerts, not just appearances
		}
	}

	// Convert to slice and sort by frequency
	var frequencies []LocationFrequency
	for location, count := range locationCount {
		frequencies = append(frequencies, LocationFrequency{
			Location: location,
			Count:    count,
		})
	}

	sort.Slice(frequencies, func(i, j int) bool {
		return frequencies[i].Count > frequencies[j].Count
	})

	// Apply limit if specified
	if limit > 0 && len(frequencies) > limit {
		frequencies = frequencies[:limit]
	}

	return frequencies
}

// GetAllArtistsSorted returns all artists sorted alphabetically by name.
func (s *Service) GetAllArtistsSorted() []models.Artist {
	artists := s.store.GetAllArtists()
	return s.sortArtistsByName(artists)
}

// SearchArtists searches for artists by name or member names (case-insensitive).
// Returns artists sorted alphabetically by name.
func (s *Service) SearchArtists(query string) []models.Artist {
	allArtists := s.store.GetAllArtists()

	if query == "" {
		return s.sortArtistsByName(allArtists)
	}

	query = strings.ToLower(query)
	var results []models.Artist

	for _, artist := range allArtists {
		// Search in artist name
		if strings.Contains(strings.ToLower(artist.Name), query) {
			results = append(results, artist)
			continue
		}

		// Search in member names
		found := false
		for _, member := range artist.Members {
			if strings.Contains(strings.ToLower(member), query) {
				results = append(results, artist)
				found = true
				break
			}
		}
		if found {
			continue
		}
	}

	return s.sortArtistsByName(results)
}

// FilterArtistsByYear filters artists by creation year range.
// If minYear or maxYear is 0, that bound is ignored.
// Returns artists sorted alphabetically by name.
func (s *Service) FilterArtistsByYear(minYear, maxYear int) []models.Artist {
	allArtists := s.store.GetAllArtists()
	var results []models.Artist

	for _, artist := range allArtists {
		// If no year restrictions, include all
		if minYear == 0 && maxYear == 0 {
			results = append(results, artist)
			continue
		}

		// Apply year filters
		if minYear > 0 && artist.CreationYear < minYear {
			continue
		}
		if maxYear > 0 && artist.CreationYear > maxYear {
			continue
		}

		results = append(results, artist)
	}

	return s.sortArtistsByName(results)
}

// GetUniqueLocationsSorted returns unique locations sorted alphabetically.
func (s *Service) GetUniqueLocationsSorted() []string {
	locations := s.store.GetUniqueLocations()
	sorted := make([]string, len(locations))
	copy(sorted, locations)
	sort.Strings(sorted)
	return sorted
}

// GetUniqueDatesSorted returns unique dates sorted.
func (s *Service) GetUniqueDatesSorted() []string {
	dates := s.store.GetUniqueDates()
	sorted := make([]string, len(dates))
	copy(sorted, dates)
	sort.Strings(sorted)
	return sorted
}

// GetStats returns comprehensive statistics about the stored data.
func (s *Service) GetStats() map[string]int {
	allArtists := s.store.GetAllArtists()
	allRelations := s.store.GetAllRelations()
	locations := s.store.GetUniqueLocations()
	dates := s.store.GetUniqueDates()

	// Calculate total concerts
	totalConcerts := 0
	locationConcerts := make(map[string]int)

	for _, relation := range allRelations {
		for location, concertDates := range relation.DatesLocations {
			concertCount := len(concertDates)
			totalConcerts += concertCount
			locationConcerts[location] += concertCount
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

// CalculateTotalShows calculates the total number of shows for an artist
func (s *Service) CalculateTotalShows(relation models.Relation) int {
	total := 0
	for _, dates := range relation.DatesLocations {
		total += len(dates)
	}
	return total
}

// ExtractCountries extracts unique countries from relation data
func (s *Service) ExtractCountries(relation models.Relation) []string {
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

// sortArtistsByName sorts artists alphabetically by name (case-insensitive).
func (s *Service) sortArtistsByName(artists []models.Artist) []models.Artist {
	// Create a copy to avoid modifying the original slice
	sorted := make([]models.Artist, len(artists))
	copy(sorted, artists)

	sort.Slice(sorted, func(i, j int) bool {
		return strings.ToLower(sorted[i].Name) < strings.ToLower(sorted[j].Name)
	})

	return sorted
}

// GetLocationDetailsBySlug returns detailed information about a specific location
func (s *Service) GetLocationDetailsBySlug(locationSlug string) (*LocationStat, bool) {
	// Get all location stats
	locationStats := s.CalculateLocationStats()

	// Find the matching location by comparing slugs
	for _, stat := range locationStats {
		if models.GenerateLocationSlug(stat.Name) == locationSlug {
			return &stat, true
		}
	}

	return nil, false
}

// GetLocationConcertDates returns all concert dates for a specific location
func (s *Service) GetLocationConcertDates(locationName string) []string {
	var allDates []string
	allRelations := s.store.GetAllRelations()

	// Collect all dates for this location from all relations
	for _, relation := range allRelations {
		if dates, exists := relation.DatesLocations[locationName]; exists {
			allDates = append(allDates, dates...)
		}
	}

	// Remove duplicates and sort
	dateMap := make(map[string]bool)
	for _, date := range allDates {
		dateMap[date] = true
	}

	uniqueDates := make([]string, 0, len(dateMap))
	for date := range dateMap {
		uniqueDates = append(uniqueDates, date)
	}

	sort.Strings(uniqueDates)
	return uniqueDates
}

// GetArtistsWithDatesForLocation returns a slice of ArtistWithDates for a specific location.
// Each entry contains the artist and the sorted, unique dates they performed at the location.
func (s *Service) GetArtistsWithDatesForLocation(locationName string) []ArtistWithDates {
	allRelations := s.store.GetAllRelations()
	// Map of artist ID to dates
	artistDates := make(map[int]map[string]bool)
	// Keep order of artists as they appear in store's artist list for determinism
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
	// Build artist lookup
	artistMap := make(map[int]models.Artist)
	for _, a := range s.store.GetAllArtists() {
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
