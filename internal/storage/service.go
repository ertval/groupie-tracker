// Package storage provides data manipulation and service operations for Groupie Tracker.
// This file contains search, filtering, ordering, and other data manipulation operations.
package storage

import (
	"sort"
	"strings"

	"groupie-tracker/internal/models"
)

// DataReader defines the interface for reading data from the base store.
type DataReader interface {
	GetAllArtists() []models.Artist
	GetArtist(id int) (models.Artist, bool)
	GetAllLocations() []models.Location
	GetLocation(id int) (models.Location, bool)
	GetAllDates() []models.Date
	GetDate(id int) (models.Date, bool)
	GetAllRelations() []models.Relation
	GetRelation(id int) (models.Relation, bool)
	GetUniqueLocations() []string
	GetUniqueDates() []string
	GetStats() map[string]int
}

// Service provides advanced data manipulation operations on top of a base data store.
// It handles searching, filtering, ordering, and other complex queries.
type Service struct {
	store DataReader
}

// NewService creates a new service instance with the given data store.
func NewService(store DataReader) *Service {
	return &Service{
		store: store,
	}
}

// SearchArtists searches for artists by name or member names (case-insensitive).
// Returns artists sorted alphabetically by name.
func (s *Service) SearchArtists(query string) []models.Artist {
	allArtists := s.store.GetAllArtists()

	if query == "" {
		return s.SortArtistsByName(allArtists)
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

	return s.SortArtistsByName(results)
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

	return s.SortArtistsByName(results)
}

// FilterArtistsByMemberCount filters artists by the number of members.
// If exact is true, returns artists with exactly memberCount members.
// If exact is false, returns artists with at least memberCount members.
func (s *Service) FilterArtistsByMemberCount(memberCount int, exact bool) []models.Artist {
	allArtists := s.store.GetAllArtists()
	var results []models.Artist

	for _, artist := range allArtists {
		if exact {
			if len(artist.Members) == memberCount {
				results = append(results, artist)
			}
		} else {
			if len(artist.Members) >= memberCount {
				results = append(results, artist)
			}
		}
	}

	return s.SortArtistsByName(results)
}

// SearchArtistsByLocation searches for artists who have performed at locations matching the query.
// Returns artists sorted alphabetically by name.
func (s *Service) SearchArtistsByLocation(query string) []models.Artist {
	if query == "" {
		return s.SortArtistsByName(s.store.GetAllArtists())
	}

	query = strings.ToLower(query)
	allRelations := s.store.GetAllRelations()
	allArtists := s.store.GetAllArtists()

	// Create a map of artist IDs that match the location query
	matchingArtistIDs := make(map[int]bool)

	for _, relation := range allRelations {
		for location := range relation.DatesLocations {
			if strings.Contains(strings.ToLower(location), query) {
				matchingArtistIDs[relation.ID] = true
				break
			}
		}
	}

	var results []models.Artist
	for _, artist := range allArtists {
		if matchingArtistIDs[artist.ID] {
			results = append(results, artist)
		}
	}

	return s.SortArtistsByName(results)
}

// SortArtistsByName sorts artists alphabetically by name (case-insensitive).
func (s *Service) SortArtistsByName(artists []models.Artist) []models.Artist {
	// Create a copy to avoid modifying the original slice
	sorted := make([]models.Artist, len(artists))
	copy(sorted, artists)

	sort.Slice(sorted, func(i, j int) bool {
		return strings.ToLower(sorted[i].Name) < strings.ToLower(sorted[j].Name)
	})

	return sorted
}

// SortArtistsByYear sorts artists by creation year (oldest first).
func (s *Service) SortArtistsByYear(artists []models.Artist) []models.Artist {
	// Create a copy to avoid modifying the original slice
	sorted := make([]models.Artist, len(artists))
	copy(sorted, artists)

	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].CreationYear < sorted[j].CreationYear
	})

	return sorted
}

// SortArtistsByMemberCount sorts artists by member count (ascending).
func (s *Service) SortArtistsByMemberCount(artists []models.Artist) []models.Artist {
	// Create a copy to avoid modifying the original slice
	sorted := make([]models.Artist, len(artists))
	copy(sorted, artists)

	sort.Slice(sorted, func(i, j int) bool {
		return len(sorted[i].Members) < len(sorted[j].Members)
	})

	return sorted
}

// GetArtistsByYearRange returns artists created within a specific year range.
// Returns artists sorted by creation year.
func (s *Service) GetArtistsByYearRange(startYear, endYear int) []models.Artist {
	filtered := s.FilterArtistsByYear(startYear, endYear)
	return s.SortArtistsByYear(filtered)
}

// GetMostPopularLocations returns the most frequently used concert locations.
// Returns up to 'limit' locations sorted by frequency (most frequent first).
func (s *Service) GetMostPopularLocations(limit int) []LocationFrequency {
	allRelations := s.store.GetAllRelations()
	locationCount := make(map[string]int)

	// Count occurrences of each location
	for _, relation := range allRelations {
		for location := range relation.DatesLocations {
			locationCount[location]++
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

// GetDetailedStats returns comprehensive statistics about the data.
func (s *Service) GetDetailedStats() DetailedStats {
	baseStats := s.store.GetStats()
	allArtists := s.store.GetAllArtists()

	var totalMembers int
	var oldestYear, newestYear int
	memberCounts := make(map[int]int)

	for i, artist := range allArtists {
		totalMembers += len(artist.Members)
		memberCounts[len(artist.Members)]++

		if i == 0 {
			oldestYear = artist.CreationYear
			newestYear = artist.CreationYear
		} else {
			if artist.CreationYear < oldestYear {
				oldestYear = artist.CreationYear
			}
			if artist.CreationYear > newestYear {
				newestYear = artist.CreationYear
			}
		}
	}

	var avgMembers float64
	if len(allArtists) > 0 {
		avgMembers = float64(totalMembers) / float64(len(allArtists))
	}

	return DetailedStats{
		BasicStats:           baseStats,
		TotalMembers:         totalMembers,
		AverageMembers:       avgMembers,
		OldestArtistYear:     oldestYear,
		NewestArtistYear:     newestYear,
		MemberCountBreakdown: memberCounts,
	}
}

// LocationFrequency represents a location and how frequently it's used for concerts.
type LocationFrequency struct {
	Location string `json:"location"`
	Count    int    `json:"count"`
}

// DetailedStats provides comprehensive statistics about the data.
type DetailedStats struct {
	BasicStats           map[string]int `json:"basic_stats"`
	TotalMembers         int            `json:"total_members"`
	AverageMembers       float64        `json:"average_members"`
	OldestArtistYear     int            `json:"oldest_artist_year"`
	NewestArtistYear     int            `json:"newest_artist_year"`
	MemberCountBreakdown map[int]int    `json:"member_count_breakdown"`
}
