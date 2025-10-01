package data

import (
	"context"
	"fmt"
	"sync"

	"groupie-tracker/internal/api"
)

// Store holds the immutable in-memory data for the application.
// After Load() completes, all fields are read-only and thread-safe.
type Store struct {
	// API client for external data fetching
	apiClient *api.Client

	// Configuration
	withCache bool

	// Cache status
	cacheEnabled bool

	// Core data collections (immutable after Load)
	artists         []Artist
	artistsByID     map[int]Artist
	artistsBySlug   map[string]Artist
	locations       []Location
	locationsBySlug map[string]Location
	appStats        AppStats
	suggestions     []SearchSuggestion
	artistFilters   ArtistFilterOptions
	locationFilters LocationFilterOptions

	// Ensure Load is called only once
	loadOnce sync.Once
	loadErr  error
}

// NewStore creates a new Store instance with the provided API client.
func NewStore(apiClient *api.Client, withCache bool) *Store {
	return &Store{
		apiClient: apiClient,
		withCache: withCache,
	}
}

// Load fetches and processes all data from the API.
// This method is thread-safe and will only execute once.
func (s *Store) Load(ctx context.Context) error {
	s.loadOnce.Do(func() {
		s.loadErr = s.loadData(ctx)
	})
	return s.loadErr
}

// loadData performs the actual data loading (called once by Load)
func (s *Store) loadData(ctx context.Context) error {
	// Fetch raw data from API concurrently using goroutines
	artistsCh := make(chan struct {
		data []api.Artist
		err  error
	}, 1)
	relationsCh := make(chan struct {
		data api.Relation
		err  error
	}, 1)

	// Fetch artists in parallel
	go func() {
		data, err := s.apiClient.FetchArtists(ctx)
		artistsCh <- struct {
			data []api.Artist
			err  error
		}{data, err}
	}()

	// Fetch relations in parallel
	go func() {
		data, err := s.apiClient.FetchRelations(ctx)
		relationsCh <- struct {
			data api.Relation
			err  error
		}{data, err}
	}()

	// Wait for both fetches to complete and check for errors
	artistsResult := <-artistsCh
	relationsResult := <-relationsCh

	if artistsResult.err != nil {
		return fmt.Errorf("failed to fetch artists: %w", artistsResult.err)
	}
	if relationsResult.err != nil {
		return fmt.Errorf("failed to fetch relations: %w", relationsResult.err)
	}

	artists := s.processArtists(artistsResult.data, relationsResult.data)

	var cachedImages, downloadedImages int
	if s.withCache {
		var cacheEnabled bool
		cacheEnabled, cachedImages, downloadedImages = s.cacheImages(artists)
		s.cacheEnabled = cacheEnabled
	} else {
		s.cacheEnabled = false
	}

	s.artists = artists

	var (
		artistsByID     map[int]Artist
		artistsBySlug   map[string]Artist
		locations       []Location
		locationsBySlug map[string]Location
		artistFilters   ArtistFilterOptions
		locationFilters LocationFilterOptions
		suggestions     []SearchSuggestion
	)

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		artistsByID, artistsBySlug = s.createArtistIndexes(artists)
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		locations, locationsBySlug = s.createLocationsData(artists)
		locationFilters = s.calculateLocationFilterOptions(locations)
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		artistFilters = s.calculateArtistFilterOptions(artists)
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		suggestions = s.generateSearchSuggestions(artists)
	}()

	wg.Wait()

	s.artistsByID = artistsByID
	s.artistsBySlug = artistsBySlug
	s.locations = locations
	s.locationsBySlug = locationsBySlug
	s.artistFilters = artistFilters
	s.locationFilters = locationFilters
	s.suggestions = suggestions
	s.appStats = s.calculateStats(artists, locations, cachedImages, downloadedImages)

	return nil
}

// Read-only accessors

// Artists returns all artists sorted by name.
func (s *Store) Artists() []Artist {
	return s.artists
}

// ArtistByID returns an artist by ID, or false if not found.
func (s *Store) ArtistByID(id int) (Artist, bool) {
	artist, ok := s.artistsByID[id]
	return artist, ok
}

// ArtistBySlug returns an artist by URL slug, or false if not found.
func (s *Store) ArtistBySlug(slug string) (Artist, bool) {
	artist, ok := s.artistsBySlug[slug]
	return artist, ok
}

// Locations returns all locations sorted by concert count.
func (s *Store) Locations() []Location {
	return s.locations
}

// LocationBySlug returns a location by URL slug, or false if not found.
func (s *Store) LocationBySlug(slug string) (Location, bool) {
	location, ok := s.locationsBySlug[slug]
	return location, ok
}

// Stats returns application statistics.
func (s *Store) Stats() AppStats {
	return s.appStats
}

// CacheEnabled returns whether image caching is enabled and functional.
func (s *Store) CacheEnabled() bool {
	return s.cacheEnabled
}

// Suggestions returns the precomputed search suggestions for autocomplete.
func (s *Store) Suggestions() []SearchSuggestion {
	return s.suggestions
}

// ArtistFilterOptions returns the precomputed artist filter metadata.
func (s *Store) ArtistFilterOptions() ArtistFilterOptions {
	return s.artistFilters
}

// LocationFilterOptions returns the precomputed location filter metadata.
func (s *Store) LocationFilterOptions() LocationFilterOptions {
	return s.locationFilters
}
