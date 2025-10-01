package domain

import (
	"context"

	"groupie-tracker/internal/api"
)

// Repository provides centralized data management for the Groupie Tracker application.
// This is a compatibility layer that wraps the Store for backward compatibility.
// New code should prefer using Store directly.
type Repository struct {
	store *Store
	// Re-expose Store fields for backward compatibility with filtering/search/test code
	artists         []Artist
	locations       []Location
	artistsByID     map[int]Artist
	artistsBySlug   map[string]Artist
	locationsBySlug map[string]Location
	cacheEnabled    bool
}

// NewRepository creates a new repository instance with the provided API client.
func NewRepository(apiClient *api.Client, withCache bool) *Repository {
	repo := &Repository{
		store: NewStore(apiClient, withCache),
	}
	return repo
}

// LoadData orchestrates the complete data loading and processing pipeline.
func (r *Repository) LoadData(ctx context.Context) error {
	err := r.store.Load(ctx)
	if err != nil {
		return err
	}
	// Update exposed fields for backward compatibility
	r.syncFromStore()
	return nil
}

// syncFromStore updates the Repository's exposed fields from the Store.
func (r *Repository) syncFromStore() {
	r.artists = r.store.Artists()
	r.locations = r.store.Locations()
	r.artistsByID = r.store.artistsByID
	r.artistsBySlug = r.store.artistsBySlug
	r.locationsBySlug = r.store.locationsBySlug
	r.cacheEnabled = r.store.CacheEnabled()
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
	artists := r.store.Artists()
	if len(artists) == 0 {
		return nil, nil
	}

	// Find the current artist index
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

	// Get previous artist (if not first)
	if currentIndex > 0 {
		prev = &artists[currentIndex-1]
	}

	// Get next artist (if not last)
	if currentIndex < len(artists)-1 {
		next = &artists[currentIndex+1]
	}

	return prev, next
}

// IsCacheEnabled returns true if image caching is enabled and functional.
func (r *Repository) IsCacheEnabled() bool {
	return r.store.CacheEnabled()
}

// SetTestData allows tests to populate the repository with test data.
// This is a legacy method for backward compatibility with existing tests.
func (r *Repository) SetTestData(artists []Artist, locations []Location) {
	// Bypass the Store's normal loading mechanism for testing
	r.store.artists = artists
	r.store.locations = locations

	// Build indexes
	r.store.artistsByID = make(map[int]Artist)
	r.store.artistsBySlug = make(map[string]Artist)
	for _, artist := range artists {
		r.store.artistsByID[artist.ID] = artist
		r.store.artistsBySlug[artist.Slug] = artist
	}

	r.store.locationsBySlug = make(map[string]Location)
	for _, location := range locations {
		r.store.locationsBySlug[location.Slug] = location
	}

	// Mock stats
	r.store.appStats = AppStats{
		TotalArtists:     len(artists),
		TotalMembers:     0,
		TotalLocations:   len(locations),
		TotalConcerts:    0,
		TotalCountries:   0,
		CachedImages:     0,
		DownloadedImages: 0,
	}

	// Sync to Repository fields
	r.syncFromStore()
}

// convertCountriesMapToSlice is a helper for tests (backward compatibility).
func (r *Repository) convertCountriesMapToSlice(countriesMap map[string]bool) []string {
	return r.store.convertCountriesMapToSlice(countriesMap)
}
