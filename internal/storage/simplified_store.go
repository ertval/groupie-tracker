// Package storage provides a simplified, unified data store for Groupie Tracker data.
package storage

import (
	"context"
	"log"
	"sort"
	"strings"
	"sync"

	"groupie-tracker/internal/models"
)

// SimplifiedStore represents a unified data store that handles all data operations
// and business logic in a single, clear structure.
type SimplifiedStore struct {
	mu sync.RWMutex

	// Core data maps
	artists     map[int]models.Artist
	artistSlugs map[string]int // slug -> artist ID mapping
	locations   map[int]models.Location
	dates       map[int]models.Date
	relations   map[int]models.Relation

	// Pre-computed data for performance
	uniqueLocations []string
	uniqueDates     []string

	// Optional API client for cache functionality
	apiClient APIClient
}

// NewSimplifiedStore creates a new empty simplified store.
func NewSimplifiedStore() *SimplifiedStore {
	return &SimplifiedStore{
		artists:         make(map[int]models.Artist),
		artistSlugs:     make(map[string]int),
		locations:       make(map[int]models.Location),
		dates:           make(map[int]models.Date),
		relations:       make(map[int]models.Relation),
		uniqueLocations: make([]string, 0),
		uniqueDates:     make([]string, 0),
	}
}

// NewSimplifiedStoreWithCache creates a new simplified store with cache functionality.
func NewSimplifiedStoreWithCache(apiClient APIClient) *SimplifiedStore {
	store := NewSimplifiedStore()
	store.apiClient = apiClient
	return store
}

// LoadData loads all data from an APIResponse into the store.
func (s *SimplifiedStore) LoadData(data models.APIResponse) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Clear existing data
	s.artists = make(map[int]models.Artist)
	s.artistSlugs = make(map[string]int)
	s.locations = make(map[int]models.Location)
	s.dates = make(map[int]models.Date)
	s.relations = make(map[int]models.Relation)

	// Load artists and generate slugs
	for _, artist := range data.Artists {
		artist.SetSlug() // Generate slug for SEO-friendly URLs
		s.artists[artist.ID] = artist
		s.artistSlugs[artist.GetSlug()] = artist.ID
	}

	// Load locations
	for _, location := range data.Locations {
		s.locations[location.ID] = location
	}

	// Load dates
	for _, date := range data.Dates {
		s.dates[date.ID] = date
	}

	// Load relations
	for _, relation := range data.Relations {
		s.relations[relation.ID] = relation
	}

	// Pre-compute unique locations and dates for performance
	s.computeUniqueData()

	log.Printf("✅ Simplified store loaded: %d artists, %d locations, %d dates, %d relations",
		len(s.artists), len(s.locations), len(s.dates), len(s.relations))
}

// RefreshData fetches fresh data from the API and reloads the store.
func (s *SimplifiedStore) RefreshData(ctx context.Context) error {
	if s.apiClient == nil {
		return nil // No API client configured
	}

	data, err := s.apiClient.FetchAllData(ctx)
	if err != nil {
		return err
	}

	s.LoadData(*data)
	return nil
}

// GetAllArtists returns all artists sorted alphabetically by name.
func (s *SimplifiedStore) GetAllArtists() []models.Artist {
	s.mu.RLock()
	defer s.mu.RUnlock()

	artists := make([]models.Artist, 0, len(s.artists))
	for _, artist := range s.artists {
		artists = append(artists, artist)
	}

	return s.sortArtistsByName(artists)
}

// GetArtist retrieves an artist by ID.
func (s *SimplifiedStore) GetArtist(id int) (models.Artist, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	artist, exists := s.artists[id]
	return artist, exists
}

// GetArtistBySlug retrieves an artist by their slug.
func (s *SimplifiedStore) GetArtistBySlug(slug string) (models.Artist, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	artistID, exists := s.artistSlugs[slug]
	if !exists {
		return models.Artist{}, false
	}

	artist, exists := s.artists[artistID]
	return artist, exists
}

// GetAllLocations returns all locations.
func (s *SimplifiedStore) GetAllLocations() []models.Location {
	s.mu.RLock()
	defer s.mu.RUnlock()

	locations := make([]models.Location, 0, len(s.locations))
	for _, location := range s.locations {
		locations = append(locations, location)
	}

	return locations
}

// GetLocation retrieves a location by ID.
func (s *SimplifiedStore) GetLocation(id int) (models.Location, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	location, exists := s.locations[id]
	return location, exists
}

// GetAllDates returns all dates.
func (s *SimplifiedStore) GetAllDates() []models.Date {
	s.mu.RLock()
	defer s.mu.RUnlock()

	dates := make([]models.Date, 0, len(s.dates))
	for _, date := range s.dates {
		dates = append(dates, date)
	}

	return dates
}

// GetDate retrieves a date by ID.
func (s *SimplifiedStore) GetDate(id int) (models.Date, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	date, exists := s.dates[id]
	return date, exists
}

// GetAllRelations returns all relations.
func (s *SimplifiedStore) GetAllRelations() []models.Relation {
	s.mu.RLock()
	defer s.mu.RUnlock()

	relations := make([]models.Relation, 0, len(s.relations))
	for _, relation := range s.relations {
		relations = append(relations, relation)
	}

	return relations
}

// GetRelation retrieves a relation by ID.
func (s *SimplifiedStore) GetRelation(id int) (models.Relation, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	relation, exists := s.relations[id]
	return relation, exists
}

// GetUniqueLocations returns a slice of unique location strings.
func (s *SimplifiedStore) GetUniqueLocations() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Return a copy to prevent external modification
	result := make([]string, len(s.uniqueLocations))
	copy(result, s.uniqueLocations)
	return result
}

// GetUniqueDates returns a slice of unique date strings.
func (s *SimplifiedStore) GetUniqueDates() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Return a copy to prevent external modification
	result := make([]string, len(s.uniqueDates))
	copy(result, s.uniqueDates)
	return result
}

// GetStats returns basic statistics about the stored data.
func (s *SimplifiedStore) GetStats() map[string]int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return map[string]int{
		"artists":   len(s.artists),
		"locations": len(s.locations),
		"dates":     len(s.dates),
		"relations": len(s.relations),
	}
}

// SearchArtists searches for artists by name or member names (case-insensitive).
// Returns artists sorted alphabetically by name.
func (s *SimplifiedStore) SearchArtists(query string) []models.Artist {
	allArtists := s.GetAllArtists()

	if query == "" {
		return allArtists
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
func (s *SimplifiedStore) FilterArtistsByYear(minYear, maxYear int) []models.Artist {
	allArtists := s.GetAllArtists()
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

// computeUniqueData pre-computes unique locations and dates for performance.
// This method should be called after loading data and must be called with write lock held.
func (s *SimplifiedStore) computeUniqueData() {
	locationSet := make(map[string]bool)
	dateSet := make(map[string]bool)

	// Extract unique locations from relations
	for _, relation := range s.relations {
		for location, dates := range relation.DatesLocations {
			locationSet[location] = true
			for _, date := range dates {
				dateSet[date] = true
			}
		}
	}

	// Convert sets to sorted slices
	s.uniqueLocations = make([]string, 0, len(locationSet))
	for location := range locationSet {
		s.uniqueLocations = append(s.uniqueLocations, location)
	}
	sort.Strings(s.uniqueLocations)

	s.uniqueDates = make([]string, 0, len(dateSet))
	for date := range dateSet {
		s.uniqueDates = append(s.uniqueDates, date)
	}
	sort.Strings(s.uniqueDates)
}

// sortArtistsByName sorts artists alphabetically by name (case-insensitive).
func (s *SimplifiedStore) sortArtistsByName(artists []models.Artist) []models.Artist {
	// Create a copy to avoid modifying the original slice
	sorted := make([]models.Artist, len(artists))
	copy(sorted, artists)

	sort.Slice(sorted, func(i, j int) bool {
		return strings.ToLower(sorted[i].Name) < strings.ToLower(sorted[j].Name)
	})

	return sorted
}
