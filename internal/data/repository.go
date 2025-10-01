package data

import (
	"context"

	"groupie-tracker/internal/api"
)

// Repository provides centralized data management for the Groupie Tracker application.
// This is a compatibility layer that wraps the Store for backward compatibility.
// New code should prefer using Store directly.
type Repository struct {
	store *Store
	svc   *Service
}

// NewRepository creates a new repository instance with the provided API client.
func NewRepository(apiClient *api.Client, withCache bool) *Repository {
	store := NewStore(apiClient, withCache)
	return &Repository{
		store: store,
		svc:   newService(store),
	}
}

// LoadData orchestrates the complete data loading and processing pipeline.
func (r *Repository) LoadData(ctx context.Context) error {
	err := r.store.Load(ctx)
	if err != nil {
		return err
	}
	return nil
}

// GetArtists returns all artists sorted by name.
func (r *Repository) GetArtists() []Artist {
	return r.store.Artists()
}

// GetArtistByID returns an artist by ID with O(1) lookup.
func (r *Repository) GetArtistByID(id int) (Artist, bool) {
	return r.store.ArtistByID(id)
}

// GetArtistBySlug returns an artist by URL slug (e.g., "queen").
func (r *Repository) GetArtistBySlug(slug string) (Artist, bool) {
	return r.store.ArtistBySlug(slug)
}

// GetLocations returns all locations sorted by concert count (descending).
func (r *Repository) GetLocations() []Location {
	return r.store.Locations()
}

// GetLocationBySlug returns a location by URL slug (e.g., "london-uk").
func (r *Repository) GetLocationBySlug(slug string) (Location, bool) {
	return r.store.LocationBySlug(slug)
}

// GetAppStats returns type-safe application statistics.
func (r *Repository) GetAppStats() AppStats {
	return r.store.Stats()
}

// GetAdjacentArtists finds the previous and next artists in alphabetical order.
func (r *Repository) GetAdjacentArtists(currentID int) (prev, next *Artist) {
	return r.svc.GetAdjacentArtists(currentID)
}

// IsCacheEnabled returns true if image caching is enabled and functional.
func (r *Repository) IsCacheEnabled() bool {
	return r.store.CacheEnabled()
}

// convertCountriesMapToSlice is a helper for tests (backward compatibility).
func (r *Repository) convertCountriesMapToSlice(countriesMap map[string]bool) []string {
	return r.store.convertCountriesMapToSlice(countriesMap)
}

// FilterArtists filters artists based on the given criteria (compatibility wrapper).
func (r *Repository) FilterArtists(criteria ArtistFilterParams) []Artist {
	return r.svc.FilterArtists(criteria)
}

// FilterLocations filters locations based on the given criteria (compatibility wrapper).
func (r *Repository) FilterLocations(criteria LocationFilterParams) []Location {
	return r.svc.FilterLocations(criteria)
}

// SearchArtists searches for artists matching the query and optional filters (compatibility wrapper).
func (r *Repository) SearchArtists(params SearchParams) SearchResult {
	return r.svc.SearchArtists(params)
}

// GetArtistFilterOptions returns all available filter options for artists (compatibility wrapper).
func (r *Repository) GetArtistFilterOptions() ArtistFilterOptions {
	return r.svc.GetArtistFilterOptions()
}

// GetLocationFilterOptions returns all available filter options for locations (compatibility wrapper).
func (r *Repository) GetLocationFilterOptions() LocationFilterOptions {
	return r.svc.GetLocationFilterOptions()
}

// GenerateAllSearchSuggestions returns precomputed search suggestion data (compatibility wrapper).
func (r *Repository) GenerateAllSearchSuggestions() []SearchSuggestion {
	return r.svc.GenerateAllSearchSuggestions()
}
