package data

import (
	"context"

	"groupie-tracker/internal/api"
)

// Service provides business logic operations on top of the immutable Store.
// All methods are read-only and thread-safe after the Store is loaded.
type Service struct {
	store *Store
}

// NewService creates a new Service with the given API client and cache settings.
func NewService(apiClient *api.Client, withCache bool) *Service {
	return newService(NewStore(apiClient, withCache))
}

// newService wraps an existing store. It is unexported to keep construction controlled within the package.
func newService(store *Store) *Service {
	return &Service{store: store}
}

// Load initializes the service by fetching and processing all data.
func (s *Service) Load(ctx context.Context) error {
	return s.store.Load(ctx)
}

// Artists returns all artists sorted by name.
func (s *Service) Artists() []Artist {
	return s.store.Artists()
}

// ArtistByID returns an artist by ID.
func (s *Service) ArtistByID(id int) (Artist, bool) {
	return s.store.ArtistByID(id)
}

// ArtistBySlug returns an artist by URL slug.
func (s *Service) ArtistBySlug(slug string) (Artist, bool) {
	return s.store.ArtistBySlug(slug)
}

// Locations returns all locations sorted by concert count.
func (s *Service) Locations() []Location {
	return s.store.Locations()
}

// LocationBySlug returns a location by URL slug.
func (s *Service) LocationBySlug(slug string) (Location, bool) {
	return s.store.LocationBySlug(slug)
}

// Stats returns application statistics.
func (s *Service) Stats() AppStats {
	return s.store.Stats()
}

// CacheEnabled returns whether image caching is enabled and functional.
func (s *Service) CacheEnabled() bool {
	return s.store.CacheEnabled()
}

// GetAdjacentArtists finds the previous and next artists in alphabetical order.
func (s *Service) GetAdjacentArtists(currentID int) (prev, next *Artist) {
	artists := s.store.Artists()
	if len(artists) == 0 {
		return nil, nil
	}

	currentIndex := -1
	for i, artist := range artists {
		if artist.ID == currentID {
			currentIndex = i
			break
		}
	}

	if currentIndex == -1 {
		return nil, nil
	}

	if currentIndex > 0 {
		prev = &artists[currentIndex-1]
	}

	if currentIndex < len(artists)-1 {
		next = &artists[currentIndex+1]
	}

	return prev, next
}

// Store exposes the underlying store for legacy code paths.
func (s *Service) Store() *Store {
	return s.store
}
