// Package store provides in-memory data storage with thread-safe access patterns.
// This replaces the global variables from the original implementation with proper
// dependency injection and encapsulation.
package store

import (
	"groupie-tracker/internal/models"
	"sync"
)

// DataStore provides centralized, thread-safe data management.
// Uses "load once, read many" pattern with strategic indexing for fast lookups.
type DataStore struct {
	// mu protects concurrent access to the store
	mu sync.RWMutex

	// Core data collections
	artists   []models.Artist
	locations []models.Location
	stats     models.AppStats

	// Fast lookup indexes
	artistsByID     map[int]models.Artist
	artistsBySlug   map[string]models.Artist
	locationsBySlug map[string]models.Location

	// Search optimization
	suggestions []models.SearchSuggestion
}

// New creates a new empty DataStore with initialized maps.
func New() *DataStore {
	return &DataStore{
		artistsByID:     make(map[int]models.Artist),
		artistsBySlug:   make(map[string]models.Artist),
		locationsBySlug: make(map[string]models.Location),
		suggestions:     make([]models.SearchSuggestion, 0),
	}
}

// LoadData populates the store with processed data and builds indexes.
// This method is thread-safe and should be called once during initialization.
func (ds *DataStore) LoadData(artists []models.Artist, locations []models.Location, stats models.AppStats) {
	ds.mu.Lock()
	defer ds.mu.Unlock()

	// Store core data
	ds.artists = artists
	ds.locations = locations
	ds.stats = stats

	// Build artist indexes
	ds.artistsByID = make(map[int]models.Artist, len(artists))
	ds.artistsBySlug = make(map[string]models.Artist, len(artists))
	for _, artist := range artists {
		ds.artistsByID[artist.ID] = artist
		ds.artistsBySlug[artist.Slug] = artist
	}

	// Build location indexes
	ds.locationsBySlug = make(map[string]models.Location, len(locations))
	for _, location := range locations {
		ds.locationsBySlug[location.Slug] = location
	}
}

// LoadSuggestions stores search suggestions for autocomplete functionality.
func (ds *DataStore) LoadSuggestions(suggestions []models.SearchSuggestion) {
	ds.mu.Lock()
	defer ds.mu.Unlock()
	ds.suggestions = suggestions
}

// GetAllArtists returns a copy of all artists for safe concurrent access.
func (ds *DataStore) GetAllArtists() []models.Artist {
	ds.mu.RLock()
	defer ds.mu.RUnlock()

	// Return a copy to prevent external modification
	result := make([]models.Artist, len(ds.artists))
	copy(result, ds.artists)
	return result
}

// GetAllLocations returns a copy of all locations for safe concurrent access.
func (ds *DataStore) GetAllLocations() []models.Location {
	ds.mu.RLock()
	defer ds.mu.RUnlock()

	// Return a copy to prevent external modification
	result := make([]models.Location, len(ds.locations))
	copy(result, ds.locations)
	return result
}

// GetArtistByID returns an artist by ID with thread-safe access.
func (ds *DataStore) GetArtistByID(id int) (models.Artist, bool) {
	ds.mu.RLock()
	defer ds.mu.RUnlock()

	artist, exists := ds.artistsByID[id]
	return artist, exists
}

// GetArtistBySlug returns an artist by slug with thread-safe access.
func (ds *DataStore) GetArtistBySlug(slug string) (models.Artist, bool) {
	ds.mu.RLock()
	defer ds.mu.RUnlock()

	artist, exists := ds.artistsBySlug[slug]
	return artist, exists
}

// GetLocationBySlug returns a location by slug with thread-safe access.
func (ds *DataStore) GetLocationBySlug(slug string) (models.Location, bool) {
	ds.mu.RLock()
	defer ds.mu.RUnlock()

	location, exists := ds.locationsBySlug[slug]
	return location, exists
}

// GetStats returns a copy of the application statistics.
func (ds *DataStore) GetStats() models.AppStats {
	ds.mu.RLock()
	defer ds.mu.RUnlock()

	// AppStats is a value type, so this is automatically a copy
	return ds.stats
}

// GetSuggestions returns a copy of search suggestions for safe concurrent access.
func (ds *DataStore) GetSuggestions() []models.SearchSuggestion {
	ds.mu.RLock()
	defer ds.mu.RUnlock()

	// Return a copy to prevent external modification
	result := make([]models.SearchSuggestion, len(ds.suggestions))
	copy(result, ds.suggestions)
	return result
}

// GetArtistCount returns the total number of artists in the store.
func (ds *DataStore) GetArtistCount() int {
	ds.mu.RLock()
	defer ds.mu.RUnlock()
	return len(ds.artists)
}

// GetLocationCount returns the total number of locations in the store.
func (ds *DataStore) GetLocationCount() int {
	ds.mu.RLock()
	defer ds.mu.RUnlock()
	return len(ds.locations)
}
