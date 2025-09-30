package data

import (
	"context"
	"fmt"
	"groupie-tracker/internal/config"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// TestRepository_PointerStorage_Consistency tests that pointer-based storage
// maintains referential integrity and allows efficient lookups.
func TestRepository_PointerStorage_Consistency(t *testing.T) {
	// Create a mock server with test data
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/artists":
			w.Write([]byte(`[
				{"id": 1, "name": "Queen", "creationDate": 1970, "firstAlbum": "Queen", "members": ["Freddie Mercury", "Brian May", "Roger Taylor", "John Deacon"]},
				{"id": 2, "name": "AC/DC", "creationDate": 1973, "firstAlbum": "High Voltage", "members": ["Angus Young", "Malcolm Young"]}
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

	// Setup test config
	config.WithCache = false
	config.APIBaseURL = server.URL
	config.APIRequestTimeout = 5 * time.Second
	repo := NewRepository()

	// Load data
	err := repo.LoadData(context.Background())
	if err != nil {
		t.Fatalf("LoadData() failed: %v", err)
	}

	// Test data consistency across different access methods
	artists := repo.GetArtists()
	if len(artists) != 2 {
		t.Fatalf("Expected 2 artists, got %d", len(artists))
	}

	// Get Queen by different methods
	var queenFromSlice *Artist
	for _, artist := range artists {
		if artist.Name == "Queen" {
			queenFromSlice = artist
			break
		}
	}
	if queenFromSlice == nil {
		t.Fatal("Queen not found in artist slice")
	}

	queenByID, foundByID := repo.GetArtistByID(1)
	if !foundByID {
		t.Fatal("Queen not found by ID")
	}

	queenBySlug, foundBySlug := repo.GetArtistBySlug("queen")
	if !foundBySlug {
		t.Fatal("Queen not found by slug")
	}

	// Test that all access methods return consistent data
	if queenFromSlice.ID != queenByID.ID || queenByID.ID != queenBySlug.ID {
		t.Errorf("Inconsistent Queen ID: slice=%d, byID=%d, bySlug=%d",
			queenFromSlice.ID, queenByID.ID, queenBySlug.ID)
	}

	// Test audit invariant: Queen should have exactly 4 members in our test data
	if len(queenByID.Members) != 4 {
		t.Errorf("Queen should have 4 members, got %d", len(queenByID.Members))
	}

	// Test pointer consistency - all lookups should return pointers to the same object
	if queenFromSlice != queenByID {
		t.Error("queenFromSlice and queenByID should point to the same object")
	}
	if queenByID != queenBySlug {
		t.Error("queenByID and queenBySlug should point to the same object")
	}
}

// TestRepository_ArtistIndex_Performance tests that the artistIndex provides O(1) lookups
// for adjacent artist operations.
func TestRepository_ArtistIndex_Performance(t *testing.T) {
	// This test measures the performance improvement from using artistIndex
	// Currently GetAdjacentArtists does O(n) linear search - we want O(1)

	// Setup with mock data - create many artists to make O(n) vs O(1) difference measurable
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/artists":
			// Create 100 artists to test performance
			artists := `[`
			for i := 1; i <= 100; i++ {
				if i > 1 {
					artists += ","
				}
				artists += fmt.Sprintf(`{"id": %d, "name": "Artist-%03d", "creationDate": 1970, "firstAlbum": "Album"}`, i, i)
			}
			artists += `]`
			w.Write([]byte(artists))
		case "/api/relation":
			relations := `{"index": [`
			for i := 1; i <= 100; i++ {
				if i > 1 {
					relations += ","
				}
				relations += fmt.Sprintf(`{"id": %d, "datesLocations": {}}`, i)
			}
			relations += `]}`
			w.Write([]byte(relations))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	config.WithCache = false
	config.APIBaseURL = server.URL
	config.APIRequestTimeout = 5 * time.Second
	repo := NewRepository()

	err := repo.LoadData(context.Background())
	if err != nil {
		t.Fatalf("LoadData() failed: %v", err)
	}

	// Test that we can get adjacent artists
	// After hardening, this should use artistIndex for O(1) lookup instead of O(n) search
	testArtist, found := repo.GetArtistBySlug("artist-050")
	if !found {
		t.Fatal("Artist-050 not found")
	}

	// Test GetAdjacentArtists functionality
	prev, next := repo.GetAdjacentArtists(testArtist.ID)

	if prev == nil || prev.Name != "Artist-049" {
		t.Errorf("Expected previous artist to be Artist-049, got %v", prev)
	}

	if next == nil || next.Name != "Artist-051" {
		t.Errorf("Expected next artist to be Artist-051, got %v", next)
	}

	// Verify that O(1) artistIndex is working by checking internal consistency
	// (We can't measure performance directly in unit tests, but we can verify behavior)

	// Test that all adjacent lookups work correctly for edge cases
	firstArtist, found := repo.GetArtistBySlug("artist-001")
	if !found {
		t.Fatal("Artist-001 not found")
	}

	prevFirst, nextFirst := repo.GetAdjacentArtists(firstArtist.ID)
	if prevFirst != nil {
		t.Error("First artist should have no previous artist")
	}
	if nextFirst == nil || nextFirst.Name != "Artist-002" {
		t.Errorf("First artist's next should be Artist-002, got %v", nextFirst)
	}

	lastArtist, found := repo.GetArtistBySlug("artist-100")
	if !found {
		t.Fatal("Artist-100 not found")
	}

	prevLast, nextLast := repo.GetAdjacentArtists(lastArtist.ID)
	if prevLast == nil || prevLast.Name != "Artist-099" {
		t.Errorf("Last artist's previous should be Artist-099, got %v", prevLast)
	}
	if nextLast != nil {
		t.Error("Last artist should have no next artist")
	}
}
