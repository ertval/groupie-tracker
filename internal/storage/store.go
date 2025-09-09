// Package storage provides in-memory storage with cache functionality for Groupie Tracker data.
package storage

import (
	"groupie-tracker/internal/models"
	svc "groupie-tracker/internal/service"
)

// Store represents a unified data store that combines base storage operations
// with advanced data manipulation services.
type Store struct {
	*BaseStore
	*svc.Service
}

// NewStore creates a new empty store with service layer.
func NewStore() *Store {
	baseStore := NewBaseStore()
	service := svc.NewService(baseStore)

	return &Store{
		BaseStore: baseStore,
		Service:   service,
	}
}

// NewStoreWithCache creates a new store with cache functionality enabled.
func NewStoreWithCache(apiClient APIClient) *Store {
	baseStore := NewBaseStoreWithCache(apiClient)
	service := svc.NewService(baseStore)

	return &Store{
		BaseStore: baseStore,
		Service:   service,
	}
}

// Override SearchArtists to maintain backward compatibility and use the service layer.
// This method ensures the existing API remains intact while using the new service architecture.
func (s *Store) SearchArtists(query string) []models.Artist {
	return s.Service.SearchArtists(query)
}

// Override FilterArtistsByYear to maintain backward compatibility and use the service layer.
// This method ensures the existing API remains intact while using the new service architecture.
func (s *Store) FilterArtistsByYear(minYear, maxYear int) []models.Artist {
	return s.Service.FilterArtistsByYear(minYear, maxYear)
}

// GetAllArtists returns all artists in the store, sorted alphabetically by name for consistency.
// This overrides the BaseStore method to provide a consistent sorting behavior.
func (s *Store) GetAllArtists() []models.Artist {
	artists := s.BaseStore.GetAllArtists()
	return s.Service.SortArtistsByName(artists)
}
