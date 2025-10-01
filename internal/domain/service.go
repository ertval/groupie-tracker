package domain

import (
	"context"

	"groupie-tracker/internal/api"
)

// Service provides business logic operations on the data Store.
// All methods are read-only and thread-safe after the Store is loaded.
// For now, this delegates to Repository for backward compatibility.
type Service struct {
	repo *Repository
}

// NewService creates a new Service with the given API client and cache settings.
func NewService(apiClient *api.Client, withCache bool) *Service {
	return &Service{
		repo: NewRepository(apiClient, withCache),
	}
}

// Load initializes the service by fetching and processing all data.
func (s *Service) Load(ctx context.Context) error {
	return s.repo.LoadData(ctx)
}

// Artists returns all artists sorted by name.
func (s *Service) Artists() []Artist {
	return s.repo.GetArtists()
}

// ArtistByID returns an artist by ID.
func (s *Service) ArtistByID(id int) (Artist, bool) {
	return s.repo.GetArtistByID(id)
}

// ArtistBySlug returns an artist by URL slug.
func (s *Service) ArtistBySlug(slug string) (Artist, bool) {
	return s.repo.GetArtistBySlug(slug)
}

// Locations returns all locations sorted by concert count.
func (s *Service) Locations() []Location {
	return s.repo.GetLocations()
}

// LocationBySlug returns a location by URL slug.
func (s *Service) LocationBySlug(slug string) (Location, bool) {
	return s.repo.GetLocationBySlug(slug)
}

// Stats returns application statistics.
func (s *Service) Stats() AppStats {
	return s.repo.GetAppStats()
}

// CacheEnabled returns whether image caching is enabled and functional.
func (s *Service) CacheEnabled() bool {
	return s.repo.IsCacheEnabled()
}

// GetAdjacentArtists finds the previous and next artists in alphabetical order.
func (s *Service) GetAdjacentArtists(currentID int) (prev, next *Artist) {
	return s.repo.GetAdjacentArtists(currentID)
}

// FilterArtists filters artists based on the given criteria.
func (s *Service) FilterArtists(criteria ArtistFilterParams) []Artist {
	return s.repo.FilterArtists(criteria)
}

// FilterLocations filters locations based on the given criteria.
func (s *Service) FilterLocations(criteria LocationFilterParams) []Location {
	return s.repo.FilterLocations(criteria)
}

// SearchArtists searches for artists matching the query and optional filters.
func (s *Service) SearchArtists(params SearchParams) SearchResult {
	return s.repo.SearchArtists(params)
}

// GetArtistFilterOptions returns all available filter options for artists.
func (s *Service) GetArtistFilterOptions() ArtistFilterOptions {
	return s.repo.GetArtistFilterOptions()
}

// GetLocationFilterOptions returns all available filter options for locations.
func (s *Service) GetLocationFilterOptions() LocationFilterOptions {
	return s.repo.GetLocationFilterOptions()
}

// GenerateSearchSuggestions generates search suggestions from all data.
func (s *Service) GenerateSearchSuggestions() []SearchSuggestion {
	return s.repo.GenerateAllSearchSuggestions()
}
