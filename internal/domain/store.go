package domain

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
	var apiArtists []api.Artist
	var apiRelations api.Relation

	// Channels to receive results
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

	apiArtists = artistsResult.data
	apiRelations = relationsResult.data

	// Transform and enrich API data into domain models
	artists := s.processArtists(apiArtists, apiRelations)

	// Build indexes for fast lookups
	s.buildIndexes(artists)

	// Cache images if enabled
	if s.withCache {
		s.cacheEnabled = s.cacheImages(artists)
	}

	// Build location aggregates
	s.buildLocations()

	// Compute application statistics
	s.computeStats()

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
