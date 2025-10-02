package data

import (
	"context"
	"sync"

	"groupie-tracker/internal/api"
)

const defaultSearchCacheSize = 50

// Service provides business logic operations on top of the immutable Store.
// All methods are read-only and thread-safe after the Store is loaded.
type Service struct {
	store *Store

	cacheMu         sync.Mutex
	searchCache     map[string][]Artist
	searchOrder     []string
	searchCacheSize int
}

// NewService creates a new Service with the given API client and cache settings.
func NewService(apiClient *api.Client, withCache bool) *Service {
	return newService(NewStore(apiClient, withCache))
}

// newService wraps an existing store. It is unexported to keep construction controlled within the package.
func newService(store *Store) *Service {
	return &Service{
		store:           store,
		searchCache:     make(map[string][]Artist, defaultSearchCacheSize),
		searchOrder:     make([]string, 0, defaultSearchCacheSize),
		searchCacheSize: defaultSearchCacheSize,
	}
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

func (s *Service) getCachedSearchResults(query string) ([]Artist, bool) {
	s.cacheMu.Lock()
	defer s.cacheMu.Unlock()

	results, ok := s.searchCache[query]
	if !ok {
		return nil, false
	}

	s.moveKeyToEndLocked(query)
	return results, true
}

func (s *Service) setCachedSearchResults(query string, results []Artist) {
	s.cacheMu.Lock()
	defer s.cacheMu.Unlock()

	if s.searchCache == nil {
		s.searchCache = make(map[string][]Artist, defaultSearchCacheSize)
	}
	if s.searchCacheSize <= 0 {
		s.searchCacheSize = defaultSearchCacheSize
	}

	if _, exists := s.searchCache[query]; exists {
		s.searchCache[query] = results
		s.moveKeyToEndLocked(query)
		return
	}

	if len(s.searchOrder) >= s.searchCacheSize {
		oldest := s.searchOrder[0]
		delete(s.searchCache, oldest)
		s.searchOrder = s.searchOrder[1:]
	}

	s.searchCache[query] = results
	s.searchOrder = append(s.searchOrder, query)
}

func (s *Service) moveKeyToEndLocked(query string) {
	for i, key := range s.searchOrder {
		if key == query {
			if i == len(s.searchOrder)-1 {
				return
			}
			copy(s.searchOrder[i:], s.searchOrder[i+1:])
			s.searchOrder[len(s.searchOrder)-1] = query
			return
		}
	}

	s.searchOrder = append(s.searchOrder, query)
}
