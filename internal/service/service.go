package service

import (
	"context"
	"sync"

	"groupie-tracker/internal/api"
	"groupie-tracker/internal/data"
)

const defaultSearchCacheSize = 50

// Service provides business logic operations on top of the immutable Store.
// All methods are read-only and thread-safe after the Store is loaded.
type Service struct {
	store *data.Store

	cacheMu         sync.Mutex
	searchCache     map[string][]data.Artist
	searchOrder     []string
	searchCacheSize int
}

// New creates a new Service using the provided data store.
func New(store *data.Store) *Service {
	if store == nil {
		panic("service: store cannot be nil")
	}

	return &Service{
		store:           store,
		searchCache:     make(map[string][]data.Artist, defaultSearchCacheSize),
		searchOrder:     make([]string, 0, defaultSearchCacheSize),
		searchCacheSize: defaultSearchCacheSize,
	}
}

// NewWithClient constructs a new Store from the API client and wraps it in a Service.
func NewWithClient(apiClient *api.Client, withCache bool) *Service {
	return New(data.NewStore(apiClient, withCache))
}

// Load initializes the service by fetching and processing all data.
func (s *Service) Load(ctx context.Context) error {
	return s.store.Load(ctx)
}

// Store exposes the underlying store for legacy code paths.
func (s *Service) Store() *data.Store {
	return s.store
}

// Artists returns all artists sorted by name.
func (s *Service) Artists() []data.Artist {
	return s.store.Artists()
}

// ArtistByID returns an artist by ID.
func (s *Service) ArtistByID(id int) (data.Artist, bool) {
	return s.store.ArtistByID(id)
}

// ArtistBySlug returns an artist by URL slug.
func (s *Service) ArtistBySlug(slug string) (data.Artist, bool) {
	return s.store.ArtistBySlug(slug)
}

// Locations returns all locations sorted by concert count.
func (s *Service) Locations() []data.Location {
	return s.store.Locations()
}

// LocationBySlug returns a location by URL slug.
func (s *Service) LocationBySlug(slug string) (data.Location, bool) {
	return s.store.LocationBySlug(slug)
}

// Stats returns application statistics.
func (s *Service) Stats() data.AppStats {
	return s.store.Stats()
}

// CacheEnabled returns whether image caching is enabled and functional.
func (s *Service) CacheEnabled() bool {
	return s.store.CacheEnabled()
}

// GetAdjacentArtists finds the previous and next artists in alphabetical order.
func (s *Service) GetAdjacentArtists(currentID int) (prev, next *data.Artist) {
	index, ok := s.store.ArtistPosition(currentID)
	if !ok {
		return nil, nil
	}

	artists := s.store.Artists()
	if len(artists) == 0 {
		return nil, nil
	}

	if index > 0 {
		prev = &artists[index-1]
	}

	if index < len(artists)-1 {
		next = &artists[index+1]
	}

	return prev, next
}

func (s *Service) getCachedSearchResults(query string) ([]data.Artist, bool) {
	s.cacheMu.Lock()
	defer s.cacheMu.Unlock()

	results, ok := s.searchCache[query]
	if !ok {
		return nil, false
	}

	s.moveKeyToEndLocked(query)
	return results, true
}

func (s *Service) setCachedSearchResults(query string, results []data.Artist) {
	s.cacheMu.Lock()
	defer s.cacheMu.Unlock()

	if s.searchCache == nil {
		s.searchCache = make(map[string][]data.Artist, defaultSearchCacheSize)
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
