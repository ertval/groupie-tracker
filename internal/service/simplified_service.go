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
	SearchArtists(query string) []models.Artist
}

// SimplifiedService provides business logic operations for Groupie Tracker data.
type SimplifiedService struct {
	store DataStore
}

// LocationStat represents statistics for a location.
type LocationStat struct {
	Name         string
	ArtistCount  int
	ConcertCount int
	Artists      []models.Artist
}

// NewSimplifiedService creates a new simplified service instance.
func NewSimplifiedService(store DataStore) *SimplifiedService {
	return &SimplifiedService{
		store: store,
	}
}

// CalculateLocationStats calculates statistics for each location.
func (s *SimplifiedService) CalculateLocationStats() []LocationStat {
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

	// Convert map to slice
	var locationStats []LocationStat
	for _, stat := range locationMap {
		locationStats = append(locationStats, *stat)
	}

	return locationStats
}

// SortLocationStatsByConcertCount sorts location statistics by concert count in descending order.
func (s *SimplifiedService) SortLocationStatsByConcertCount(stats []LocationStat) []LocationStat {
	// Create a copy to avoid modifying the original slice
	sorted := make([]LocationStat, len(stats))
	copy(sorted, stats)

	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].ConcertCount > sorted[j].ConcertCount
	})

	return sorted
}

// CalculateTotalCountries calculates the total number of unique countries.
func (s *SimplifiedService) CalculateTotalCountries(locationStats []LocationStat) int {
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
func (s *SimplifiedService) CalculateTotalConcerts() int {
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
func (s *SimplifiedService) GetMostPopularLocations(limit int) []LocationFrequency {
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
