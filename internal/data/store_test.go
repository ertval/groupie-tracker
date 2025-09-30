package data

import (
	"context"
	"testing"

	"groupie-tracker/internal/testsupport"
)

func TestLoadBuildsIndexes(t *testing.T) {
	loader := testsupport.MinimalDataset()

	store, err := Load(context.Background(), loader)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	if got, want := len(store.Artists()), len(loader.Artists); got != want {
		t.Fatalf("unexpected artist count: got %d want %d", got, want)
	}

	artist, ok := store.ArtistBySlug("the-example")
	if !ok {
		t.Fatalf("artist lookup by slug failed")
	}

	if artist.ConcertCount != len(artist.Concerts) {
		t.Errorf("concert count mismatch: got %d concerts %d", artist.ConcertCount, len(artist.Concerts))
	}

	if _, ok := store.ArtistByID(2); !ok {
		t.Fatalf("artist lookup by ID failed")
	}

	locations := store.Locations()
	if len(locations) == 0 {
		t.Fatalf("expected locations to be derived")
	}

	if _, ok := store.LocationBySlug("new-york-usa"); !ok {
		t.Errorf("expected location slug to exist")
	}

	stats := store.Stats()
	if stats.TotalArtists != len(loader.Artists) {
		t.Errorf("stats total artists = %d, want %d", stats.TotalArtists, len(loader.Artists))
	}
	if stats.TotalConcerts == 0 {
		t.Errorf("expected total concerts > 0")
	}
}
