// Package storage provides a unified data store for Groupie Tracker data.
package storage

import (
	"context"
	"log"
	"sync"

	"groupie-tracker/internal/models"
)

// APIClient defines the interface for fetching data from external APIs.
type APIClient interface {
	FetchAllData(ctx context.Context) (*models.APIResponse, error)
}

// Store represents a unified data store that handles all data operations
// in a single, clear structure.
type Store struct {
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

// NewStore creates a new empty store.
func NewStore() *Store {
	return &Store{
		artists:         make(map[int]models.Artist),
		artistSlugs:     make(map[string]int),
		locations:       make(map[int]models.Location),
		dates:           make(map[int]models.Date),
		relations:       make(map[int]models.Relation),
		uniqueLocations: make([]string, 0),
		uniqueDates:     make([]string, 0),
	}
}

// NewStoreWithCache creates a new store with cache functionality.
func NewStoreWithCache(apiClient APIClient) *Store {
	store := NewStore()
	store.apiClient = apiClient
	return store
}

// LoadData loads all data from an APIResponse into the store.
func (s *Store) LoadData(data models.APIResponse) {
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

	log.Printf("✅ Store loaded: %d artists, %d locations, %d dates, %d relations",
		len(s.artists), len(s.uniqueLocations), len(s.uniqueDates), len(s.relations))
}

// RefreshData fetches fresh data from the API and reloads the store.
func (s *Store) RefreshData(ctx context.Context) error {
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

// GetAllArtists returns all artists (raw data, no sorting).
func (s *Store) GetAllArtists() []models.Artist {
	s.mu.RLock()
	defer s.mu.RUnlock()

	artists := make([]models.Artist, 0, len(s.artists))
	for _, artist := range s.artists {
		artists = append(artists, artist)
	}

	return artists
}

// GetArtist retrieves an artist by ID.
func (s *Store) GetArtist(id int) (models.Artist, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	artist, exists := s.artists[id]
	return artist, exists
}

// GetArtistBySlug retrieves an artist by their slug.
func (s *Store) GetArtistBySlug(slug string) (models.Artist, bool) {
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
func (s *Store) GetAllLocations() []models.Location {
	s.mu.RLock()
	defer s.mu.RUnlock()

	locations := make([]models.Location, 0, len(s.locations))
	for _, location := range s.locations {
		locations = append(locations, location)
	}

	return locations
}

// GetLocation retrieves a location by ID.
func (s *Store) GetLocation(id int) (models.Location, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	location, exists := s.locations[id]
	return location, exists
}

// GetAllDates returns all dates.
func (s *Store) GetAllDates() []models.Date {
	s.mu.RLock()
	defer s.mu.RUnlock()

	dates := make([]models.Date, 0, len(s.dates))
	for _, date := range s.dates {
		dates = append(dates, date)
	}

	return dates
}

// GetDate retrieves a date by ID.
func (s *Store) GetDate(id int) (models.Date, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	date, exists := s.dates[id]
	return date, exists
}

// GetAllRelations returns all relations.
func (s *Store) GetAllRelations() []models.Relation {
	s.mu.RLock()
	defer s.mu.RUnlock()

	relations := make([]models.Relation, 0, len(s.relations))
	for _, relation := range s.relations {
		relations = append(relations, relation)
	}

	return relations
}

// GetRelation retrieves a relation by ID.
func (s *Store) GetRelation(id int) (models.Relation, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	relation, exists := s.relations[id]
	return relation, exists
}

// GetUniqueLocations returns a slice of unique location strings (raw data, no sorting).
func (s *Store) GetUniqueLocations() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Return a copy to prevent external modification
	result := make([]string, len(s.uniqueLocations))
	copy(result, s.uniqueLocations)
	return result
}

// GetUniqueDates returns a slice of unique date strings (raw data, no sorting).
func (s *Store) GetUniqueDates() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Return a copy to prevent external modification
	result := make([]string, len(s.uniqueDates))
	copy(result, s.uniqueDates)
	return result
}

// computeUniqueData pre-computes unique locations and dates for performance.
// This method should be called after loading data and must be called with write lock held.
func (s *Store) computeUniqueData() {
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

	// Convert sets to slices (no sorting - let service layer handle that)
	s.uniqueLocations = make([]string, 0, len(locationSet))
	for location := range locationSet {
		s.uniqueLocations = append(s.uniqueLocations, location)
	}

	s.uniqueDates = make([]string, 0, len(dateSet))
	for date := range dateSet {
		s.uniqueDates = append(s.uniqueDates, date)
	}
}
