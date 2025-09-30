package data

import (
	"context"
	"fmt"

	"groupie-tracker/internal/api"
)

// Loader describes the dependencies required to hydrate a Store from external data sources.
type Loader interface {
	FetchArtists(ctx context.Context) ([]api.Artist, error)
	FetchRelations(ctx context.Context) ([]api.RelationIndex, error)
}

// Store provides read-only access to the aggregated datasets used by the application.
type Store struct {
	artists         []Artist
	artistsByID     map[int]Artist
	artistsBySlug   map[string]Artist
	locations       []Location
	locationsBySlug map[string]Location
	stats           AppStats
}

// Load constructs a Store instance by fetching remote data then running all transformations.
func Load(ctx context.Context, loader Loader) (*Store, error) {
	if loader == nil {
		return nil, fmt.Errorf("data: loader is required")
	}

	artistsDTO, err := loader.FetchArtists(ctx)
	if err != nil {
		return nil, fmt.Errorf("data: fetch artists: %w", err)
	}

	relationsDTO, err := loader.FetchRelations(ctx)
	if err != nil {
		return nil, fmt.Errorf("data: fetch relations: %w", err)
	}

	artists := processArtists(artistsDTO, relationsDTO)
	locations := createLocations(artists)
	stats := calculateStats(artists, locations)

	store := &Store{
		artists:         artists,
		artistsByID:     make(map[int]Artist, len(artists)),
		artistsBySlug:   make(map[string]Artist, len(artists)),
		locations:       locations,
		locationsBySlug: make(map[string]Location, len(locations)),
		stats:           stats,
	}

	for _, artist := range artists {
		store.artistsByID[artist.ID] = artist
		store.artistsBySlug[artist.Slug] = artist
	}

	for _, location := range locations {
		store.locationsBySlug[location.Slug] = location
	}

	return store, nil
}

// MustLoad mirrors Load but panics on error. Suitable for CLI/bootstrap scenarios.
func MustLoad(ctx context.Context, loader Loader) *Store {
	store, err := Load(ctx, loader)
	if err != nil {
		panic(err)
	}
	return store
}

// Artists returns all artists sorted alphabetically.
func (s *Store) Artists() []Artist {
	if s == nil {
		return nil
	}
	return s.artists
}

// ArtistByID looks up an artist by numeric identifier.
func (s *Store) ArtistByID(id int) (Artist, bool) {
	if s == nil {
		return Artist{}, false
	}
	artist, ok := s.artistsByID[id]
	return artist, ok
}

// ArtistBySlug looks up an artist by slug.
func (s *Store) ArtistBySlug(slug string) (Artist, bool) {
	if s == nil {
		return Artist{}, false
	}
	artist, ok := s.artistsBySlug[slug]
	return artist, ok
}

// Locations returns all locations sorted by concert volume.
func (s *Store) Locations() []Location {
	if s == nil {
		return nil
	}
	return s.locations
}

// LocationBySlug retrieves location details by slug.
func (s *Store) LocationBySlug(slug string) (Location, bool) {
	if s == nil {
		return Location{}, false
	}
	location, ok := s.locationsBySlug[slug]
	return location, ok
}

// Stats exposes application-wide statistics derived from the dataset.
func (s *Store) Stats() AppStats {
	if s == nil {
		return AppStats{}
	}
	return s.stats
}
