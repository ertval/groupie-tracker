// Package storage provides in-memory storage with cache functionality for Groupie Tracker data.
package storage

import (
	"context"
	"log"
	"sync"
	"sync/atomic"
	"time"

	"groupie-tracker/internal/models"
)

// ANSI color codes for pretty CLI output (standard library only)
const (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorCyan   = "\033[36m"
)

const (
	// CacheUpdateInterval is the interval at which the cache updates from the API
	CacheUpdateInterval = 30 * time.Second
)

// APIClient defines the interface for fetching data from external API
type APIClient interface {
	FetchAllData(ctx context.Context) (*models.APIResponse, error)
}

// BaseStore represents the core data store with thread-safe operations and cache functionality.
// It handles basic CRUD operations and cache management.
type BaseStore struct {
	mu        sync.RWMutex
	artists   map[int]models.Artist
	locations map[int]models.Location
	dates     map[int]models.Date
	relations map[int]models.Relation

	// Cache functionality
	apiClient      APIClient
	cacheRunning   atomic.Bool
	stopCache      chan struct{}
	cacheMu        sync.RWMutex
	lastUpdate     time.Time
	updateInterval time.Duration

	// Computed data (updated through cache)
	uniqueLocations []string
	uniqueDates     []string
}

// NewBaseStore creates a new empty base store.
func NewBaseStore() *BaseStore {
	return &BaseStore{
		artists:         make(map[int]models.Artist),
		locations:       make(map[int]models.Location),
		dates:           make(map[int]models.Date),
		relations:       make(map[int]models.Relation),
		stopCache:       make(chan struct{}),
		updateInterval:  CacheUpdateInterval,
		uniqueLocations: make([]string, 0),
		uniqueDates:     make([]string, 0),
	}
}

// NewBaseStoreWithCache creates a new base store with cache functionality enabled.
func NewBaseStoreWithCache(apiClient APIClient) *BaseStore {
	store := NewBaseStore()
	store.apiClient = apiClient
	return store
}

// StartCache begins the periodic cache update goroutine.
func (s *BaseStore) StartCache(ctx context.Context) {
	if s.apiClient == nil {
		log.Printf("Warning: Cannot start cache without API client")
		return
	}

	if s.cacheRunning.Load() {
		log.Printf("Cache already running")
		return
	}

	// Single colored start log with emoji and update interval appended.
	log.Println(colorCyan + "🔄 Starting Cache with periodic updates... " + colorReset + s.updateInterval.String() + " interval.")

	s.cacheRunning.Store(true)

	go s.cacheUpdateLoop(ctx)
}

// StopCache stops the periodic cache update goroutine.
func (s *BaseStore) StopCache() {
	if !s.cacheRunning.Load() {
		return
	}

	// Only close the stop channel and mark as not running. The higher-level
	// server will print a single stopping message; avoid duplicate logs here.
	close(s.stopCache)
	s.cacheRunning.Store(false)
}

// cacheUpdateLoop runs the periodic cache update in a goroutine.
func (s *BaseStore) cacheUpdateLoop(ctx context.Context) {
	ticker := time.NewTicker(s.updateInterval)
	defer ticker.Stop()
	defer s.cacheRunning.Store(false) // Ensure we mark as not running when loop exits

	// Initial load: perform initial fetch but do not print the periodic "Cache updated" message.
	if err := s.updateFromAPI(ctx, true); err != nil {
		log.Printf("Initial cache load failed: %v", err)
	}

	for {
		select {
		case <-ctx.Done():
			// Silent stop on context cancellation to avoid duplicate stopping messages.
			return
		case <-s.stopCache:
			// Silent stop when StopCache is called; server already prints stopping message.
			return
		case <-ticker.C:
			if err := s.updateFromAPI(ctx, false); err != nil {
				log.Printf("Cache update failed: %v", err)
			}
		}
	}
}

// updateFromAPI fetches fresh data from the API and updates the store.
// If initial is true, this is the first load and will not emit the periodic
// "Cache updated" success log (so the server can announce startup first).
func (s *BaseStore) updateFromAPI(ctx context.Context, initial bool) error {
	if s.apiClient == nil {
		return nil
	}

	data, err := s.apiClient.FetchAllData(ctx)
	if err != nil {
		return err
	}

	// data is *models.APIResponse
	s.LoadData(*data)

	s.cacheMu.Lock()
	s.lastUpdate = time.Now()
	lu := s.lastUpdate
	s.cacheMu.Unlock()

	// Only print the colorful periodic update message for non-initial updates.
	if !initial {
		log.Println(colorGreen + "✅ Cache Updated" + colorReset + " successfully at " + lu.Format(time.RFC3339))
	}
	return nil
}

// GetLastUpdate returns the timestamp of the last cache update.
func (s *BaseStore) GetLastUpdate() time.Time {
	s.cacheMu.RLock()
	defer s.cacheMu.RUnlock()
	return s.lastUpdate
}

// IsRunning returns whether the cache is currently running.
func (s *BaseStore) IsRunning() bool {
	return s.cacheRunning.Load()
}

// computeDerivedData computes unique locations and dates from the current data.
func (s *BaseStore) computeDerivedData() {
	// Compute unique location strings
	locationSet := make(map[string]bool)
	for _, location := range s.locations {
		for _, loc := range location.Locations {
			locationSet[loc] = true
		}
	}
	s.uniqueLocations = make([]string, 0, len(locationSet))
	for loc := range locationSet {
		s.uniqueLocations = append(s.uniqueLocations, loc)
	}

	// Compute unique date strings
	dateSet := make(map[string]bool)
	for _, date := range s.dates {
		for _, d := range date.Dates {
			dateSet[d] = true
		}
	}
	s.uniqueDates = make([]string, 0, len(dateSet))
	for d := range dateSet {
		s.uniqueDates = append(s.uniqueDates, d)
	}
}

// AddArtist adds an artist to the store.
func (s *BaseStore) AddArtist(artist models.Artist) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.artists[artist.ID] = artist
}

// GetArtist retrieves an artist by ID.
func (s *BaseStore) GetArtist(id int) (models.Artist, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	artist, exists := s.artists[id]
	return artist, exists
}

// GetAllArtists returns all artists in the store (unordered).
func (s *BaseStore) GetAllArtists() []models.Artist {
	s.mu.RLock()
	defer s.mu.RUnlock()

	artists := make([]models.Artist, 0, len(s.artists))
	for _, artist := range s.artists {
		artists = append(artists, artist)
	}
	return artists
}

// AddLocation adds a location to the store.
func (s *BaseStore) AddLocation(location models.Location) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.locations[location.ID] = location
}

// GetLocation retrieves a location by ID.
func (s *BaseStore) GetLocation(id int) (models.Location, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	location, exists := s.locations[id]
	return location, exists
}

// GetAllLocations returns all locations in the store.
func (s *BaseStore) GetAllLocations() []models.Location {
	s.mu.RLock()
	defer s.mu.RUnlock()

	locations := make([]models.Location, 0, len(s.locations))
	for _, location := range s.locations {
		locations = append(locations, location)
	}
	return locations
}

// GetUniqueLocations returns a slice of unique location strings.
func (s *BaseStore) GetUniqueLocations() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Return a copy to prevent external modification
	result := make([]string, len(s.uniqueLocations))
	copy(result, s.uniqueLocations)
	return result
}

// GetUniqueDates returns a slice of unique date strings.
func (s *BaseStore) GetUniqueDates() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Return a copy to prevent external modification
	result := make([]string, len(s.uniqueDates))
	copy(result, s.uniqueDates)
	return result
}

// AddDate adds a date to the store.
func (s *BaseStore) AddDate(date models.Date) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.dates[date.ID] = date
}

// GetDate retrieves a date by ID.
func (s *BaseStore) GetDate(id int) (models.Date, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	date, exists := s.dates[id]
	return date, exists
}

// GetAllDates returns all dates in the store.
func (s *BaseStore) GetAllDates() []models.Date {
	s.mu.RLock()
	defer s.mu.RUnlock()

	dates := make([]models.Date, 0, len(s.dates))
	for _, date := range s.dates {
		dates = append(dates, date)
	}
	return dates
}

// AddRelation adds a relation to the store.
func (s *BaseStore) AddRelation(relation models.Relation) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.relations[relation.ID] = relation
}

// GetRelation retrieves a relation by ID.
func (s *BaseStore) GetRelation(id int) (models.Relation, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	relation, exists := s.relations[id]
	return relation, exists
}

// GetAllRelations returns all relations in the store.
func (s *BaseStore) GetAllRelations() []models.Relation {
	s.mu.RLock()
	defer s.mu.RUnlock()

	relations := make([]models.Relation, 0, len(s.relations))
	for _, relation := range s.relations {
		relations = append(relations, relation)
	}
	return relations
}

// LoadData loads a complete dataset into the store, replacing existing data.
func (s *BaseStore) LoadData(data models.APIResponse) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Clear existing data
	s.artists = make(map[int]models.Artist)
	s.locations = make(map[int]models.Location)
	s.dates = make(map[int]models.Date)
	s.relations = make(map[int]models.Relation)

	// Load new data
	for _, artist := range data.Artists {
		s.artists[artist.ID] = artist
	}

	for _, location := range data.Locations {
		s.locations[location.ID] = location
	}

	for _, date := range data.Dates {
		s.dates[date.ID] = date
	}

	for _, relation := range data.Relations {
		s.relations[relation.ID] = relation
	}

	// Recompute derived data
	s.computeDerivedData()
}

// GetStats returns statistics about the stored data.
func (s *BaseStore) GetStats() map[string]int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return map[string]int{
		"artists":   len(s.artists),
		"locations": len(s.uniqueLocations),
		"dates":     len(s.uniqueDates),
		"relations": len(s.relations),
	}
}
