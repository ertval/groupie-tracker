package data

import (
	"context"
	"groupie-tracker/internal/config"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestRepository_LoadData_Success(t *testing.T) {
	// Create a mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/artists":
			w.Write([]byte(`[
				{"id": 1, "name": "Queen", "creationDate": 1970, "firstAlbum": "Queen"},
				{"id": 2, "name": "AC/DC", "creationDate": 1973, "firstAlbum": "High Voltage"}
			]`))
		case "/api/relation":
			w.Write([]byte(`{
				"index": [
					{"id": 1, "datesLocations": {"london-uk": ["2022-01-01"], "paris-france": ["2022-02-01"]}},
					{"id": 2, "datesLocations": {"london-uk": ["2023-01-01"]}}
				]
			}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	// Disable image caching for tests to avoid creating files on disk
	config.WithCache = false
	// Point repository to the mock server and set a short timeout for tests
	config.APIBaseURL = server.URL
	config.APIRequestTimeout = 5 * time.Second
	repo := NewRepository()

	// Load the data
	if err := repo.LoadData(context.Background()); err != nil {
		t.Fatalf("LoadData() failed: %v", err)
	}

	// --- Assertions ---

	// Check artists slice
	artists := repo.GetArtists()
	if len(artists) != 2 {
		t.Errorf("expected 2 artists, got %d", len(artists))
	}
	// Check if sorted by name (AC/DC should be first)
	if artists[0].Name != "AC/DC" {
		t.Errorf("expected first artist to be AC/DC, got %s", artists[0].Name)
	}

	// Check artist maps
	acdc, ok := repo.GetArtistBySlug("ac-dc")
	if !ok {
		t.Fatal("could not find artist by slug 'ac-dc'")
	}
	if acdc.Name != "AC/DC" {
		t.Errorf("expected name AC/DC, got %s", acdc.Name)
	}
	if acdc.ConcertCount != 1 {
		t.Errorf("expected AC/DC to have 1 concert, got %d", acdc.ConcertCount)
	}

	queen, ok := repo.GetArtistByID(1)
	if !ok {
		t.Fatal("could not find artist by ID 1")
	}
	if queen.Name != "Queen" {
		t.Errorf("expected name Queen, got %s", queen.Name)
	}
	if queen.ConcertCount != 2 {
		t.Errorf("expected Queen to have 2 concerts, got %d", queen.ConcertCount)
	}

	// Check navigation IDs
	if acdc.NextArtistID != queen.ID {
		t.Errorf("expected AC/DC's next artist to be Queen, got ID %d", acdc.NextArtistID)
	}
	if queen.PrevArtistID != acdc.ID {
		t.Errorf("expected Queen's previous artist to be AC/DC, got ID %d", queen.PrevArtistID)
	}

	// Check locations
	locations := repo.GetLocations()
	if len(locations) != 2 {
		t.Errorf("expected 2 locations, got %d", len(locations))
	}

	london, ok := repo.GetLocationBySlug("london-uk")
	if !ok {
		t.Fatal("could not find location by slug 'london-uk'")
	}
	if london.ArtistCount != 2 {
		t.Errorf("expected london to have 2 artists, got %d", london.ArtistCount)
	}
	if london.TotalConcerts != 2 {
		t.Errorf("expected london to have 2 total concerts, got %d", london.TotalConcerts)
	}

	// Check global stats
	stats := repo.GetStats()
	if stats["total_artists"] != 2 {
		t.Errorf("expected total_artists to be 2, got %d", stats["total_artists"])
	}
	if stats["total_locations"] != 2 {
		t.Errorf("expected total_locations to be 2, got %d", stats["total_locations"])
	}
	if stats["total_concerts"] != 3 {
		t.Errorf("expected total_concerts to be 3, got %d", stats["total_concerts"])
	}
}
