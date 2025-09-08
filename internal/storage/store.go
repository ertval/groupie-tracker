// Package storage provides in-memory storage for Groupie Tracker data.
package storage

import (
	"strings"
	"sync"

	"groupie-tracker/internal/models"
)

// Store represents an in-memory data store with thread-safe operations.
type Store struct {
	mu          sync.RWMutex
	artists     map[int]models.Artist
	artistSlugs map[string]int // slug -> artist ID mapping
	locations   map[int]models.Location
	dates       map[int]models.Date
	relations   map[int]models.Relation
}

// StoreData represents a complete dataset for bulk loading.
type StoreData struct {
	Artists   []models.Artist
	Locations []models.Location
	Dates     []models.Date
	Relations []models.Relation
}

// NewStore creates a new empty store.
func NewStore() *Store {
	return &Store{
		artists:     make(map[int]models.Artist),
		artistSlugs: make(map[string]int),
		locations:   make(map[int]models.Location),
		dates:       make(map[int]models.Date),
		relations:   make(map[int]models.Relation),
	}
}

// AddArtist adds an artist to the store.
func (s *Store) AddArtist(artist models.Artist) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.artists[artist.ID] = artist
}

// GetArtist retrieves an artist by ID.
func (s *Store) GetArtist(id int) (models.Artist, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	artist, exists := s.artists[id]
	return artist, exists
}

// GetArtistBySlug retrieves an artist by slug.
func (s *Store) GetArtistBySlug(slug string) (models.Artist, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	if id, exists := s.artistSlugs[slug]; exists {
		if artist, exists := s.artists[id]; exists {
			return artist, true
		}
	}
	return models.Artist{}, false
}

// GetAllArtists returns all artists in the store.
func (s *Store) GetAllArtists() []models.Artist {
	s.mu.RLock()
	defer s.mu.RUnlock()

	artists := make([]models.Artist, 0, len(s.artists))
	for _, artist := range s.artists {
		artists = append(artists, artist)
	}
	return artists
}

// SearchArtists searches for artists by name or member names (case-insensitive).
func (s *Store) SearchArtists(query string) []models.Artist {
	if query == "" {
		return s.GetAllArtists()
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	query = strings.ToLower(query)
	var results []models.Artist

	for _, artist := range s.artists {
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

	return results
}

// FilterArtistsByYear filters artists by creation year range.
// If minYear or maxYear is 0, that bound is ignored.
func (s *Store) FilterArtistsByYear(minYear, maxYear int) []models.Artist {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var results []models.Artist

	for _, artist := range s.artists {
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

	return results
}

// AddLocation adds a location to the store.
func (s *Store) AddLocation(location models.Location) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.locations[location.ID] = location
}

// GetLocation retrieves a location by ID.
func (s *Store) GetLocation(id int) (models.Location, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	location, exists := s.locations[id]
	return location, exists
}

// GetAllLocations returns all locations in the store.
func (s *Store) GetAllLocations() []models.Location {
	s.mu.RLock()
	defer s.mu.RUnlock()

	locations := make([]models.Location, 0, len(s.locations))
	for _, location := range s.locations {
		locations = append(locations, location)
	}
	return locations
}

// GetUniqueLocations returns a slice of unique location strings.
func (s *Store) GetUniqueLocations() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	locationSet := make(map[string]bool)

	for _, location := range s.locations {
		for _, loc := range location.Locations {
			locationSet[loc] = true
		}
	}

	uniqueLocations := make([]string, 0, len(locationSet))
	for loc := range locationSet {
		uniqueLocations = append(uniqueLocations, loc)
	}

	return uniqueLocations
}

// AddDate adds a date to the store.
func (s *Store) AddDate(date models.Date) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.dates[date.ID] = date
}

// GetDate retrieves a date by ID.
func (s *Store) GetDate(id int) (models.Date, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	date, exists := s.dates[id]
	return date, exists
}

// GetAllDates returns all dates in the store.
func (s *Store) GetAllDates() []models.Date {
	s.mu.RLock()
	defer s.mu.RUnlock()

	dates := make([]models.Date, 0, len(s.dates))
	for _, date := range s.dates {
		dates = append(dates, date)
	}
	return dates
}

// AddRelation adds a relation to the store.
func (s *Store) AddRelation(relation models.Relation) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.relations[relation.ID] = relation
}

// GetRelation retrieves a relation by ID.
func (s *Store) GetRelation(id int) (models.Relation, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	relation, exists := s.relations[id]
	return relation, exists
}

// GetAllRelations returns all relations in the store.
func (s *Store) GetAllRelations() []models.Relation {
	s.mu.RLock()
	defer s.mu.RUnlock()

	relations := make([]models.Relation, 0, len(s.relations))
	for _, relation := range s.relations {
		relations = append(relations, relation)
	}
	return relations
}

// LoadData loads a complete dataset into the store, replacing existing data.
func (s *Store) LoadData(data StoreData) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Clear existing data
	s.artists = make(map[int]models.Artist)
	s.artistSlugs = make(map[string]int)
	s.locations = make(map[int]models.Location)
	s.dates = make(map[int]models.Date)
	s.relations = make(map[int]models.Relation)

	// Load new data and generate slugs
	for _, artist := range data.Artists {
		// Generate slug for the artist
		artist.SetSlug()
		
		// Store artist
		s.artists[artist.ID] = artist
		
		// Map slug to artist ID
		if artist.Slug != "" {
			s.artistSlugs[artist.Slug] = artist.ID
		}
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
}

// GetStats returns statistics about the stored data.
func (s *Store) GetStats() map[string]int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return map[string]int{
		"artists":   len(s.artists),
		"locations": len(s.locations),
		"dates":     len(s.dates),
		"relations": len(s.relations),
	}
}
