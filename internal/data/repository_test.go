package data

import (
	"context"
	"groupie-tracker/internal/config"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
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

	// Check navigation works
	prevArtist, nextArtist := repo.GetAdjacentArtists(acdc.ID)
	if nextArtist == nil || nextArtist.ID != queen.ID {
		t.Error("expected AC/DC's next artist to be Queen")
	}

	prevArtist, nextArtist = repo.GetAdjacentArtists(queen.ID)
	if prevArtist == nil || prevArtist.ID != acdc.ID {
		t.Error("expected Queen's previous artist to be AC/DC")
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

func TestRepository_LoadData_Failure_InvalidJSON(t *testing.T) {
	// Create a mock server that returns invalid JSON
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/artists":
			w.Write([]byte(`invalid json`))
		case "/api/relation":
			w.Write([]byte(`{"index": []}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	config.WithCache = false
	config.APIBaseURL = server.URL
	config.APIRequestTimeout = 5 * time.Second
	repo := NewRepository()

	// Load the data - should fail with JSON error
	err := repo.LoadData(context.Background())
	if err == nil {
		t.Fatal("expected error for invalid JSON, got nil")
	}
	if !strings.Contains(err.Error(), "invalid character") {
		t.Errorf("expected JSON parsing error, got: %v", err)
	}
}

func TestRepository_LoadData_Failure_HTTPError(t *testing.T) {
	// Create a mock server that returns 500 error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}))
	defer server.Close()

	config.WithCache = false
	config.APIBaseURL = server.URL
	config.APIRequestTimeout = 5 * time.Second
	repo := NewRepository()

	// Load the data - should fail with HTTP error
	err := repo.LoadData(context.Background())
	if err == nil {
		t.Fatal("expected error for HTTP 500, got nil")
	}
	if !strings.Contains(err.Error(), "500") {
		t.Errorf("expected HTTP 500 error, got: %v", err)
	}
}

func TestRepository_LoadData_Timeout(t *testing.T) {
	// Create a mock server that delays response
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond) // Delay longer than timeout
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`[]`))
	}))
	defer server.Close()

	config.WithCache = false
	config.APIBaseURL = server.URL
	config.APIRequestTimeout = 10 * time.Millisecond // Very short timeout
	repo := NewRepository()

	// Load the data - should fail with timeout
	err := repo.LoadData(context.Background())
	if err == nil {
		t.Fatal("expected timeout error, got nil")
	}
	if !strings.Contains(err.Error(), "timeout") && !strings.Contains(err.Error(), "context deadline exceeded") {
		t.Errorf("expected timeout error, got: %v", err)
	}
}

func TestRepository_GetMethods_EmptyData(t *testing.T) {
	// Test getter methods when no data is loaded
	repo := NewRepository()

	// Test GetArtists with empty repository
	artists := repo.GetArtists()
	if artists != nil {
		t.Errorf("expected nil artists slice, got %v", artists)
	}

	// Test GetArtistByID with empty repository
	_, found := repo.GetArtistByID(1)
	if found {
		t.Error("expected GetArtistByID to return false for empty repo")
	}

	// Test GetArtistBySlug with empty repository
	_, found = repo.GetArtistBySlug("test")
	if found {
		t.Error("expected GetArtistBySlug to return false for empty repo")
	}

	// Test GetLocations with empty repository
	locations := repo.GetLocations()
	if locations != nil {
		t.Errorf("expected nil locations slice, got %v", locations)
	}

	// Test GetLocationBySlug with empty repository
	_, found = repo.GetLocationBySlug("test")
	if found {
		t.Error("expected GetLocationBySlug to return false for empty repo")
	}

	// Test GetStats with empty repository
	stats := repo.GetStats()
	if stats != nil {
		t.Errorf("expected nil stats map, got %v", stats)
	}
}

func TestRepository_LoadData_WithImageCaching(t *testing.T) {
	// Create a mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/artists":
			w.Write([]byte(`[
				{"id": 1, "name": "Test Artist", "creationDate": 2020, "firstAlbum": "Test Album", "image": "http://example.com/test.jpg"}
			]`))
		case "/api/relation":
			w.Write([]byte(`{
				"index": [
					{"id": 1, "datesLocations": {"test-location": ["2023-01-01"]}}
				]
			}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	// Enable image caching for this test
	config.WithCache = true
	config.APIBaseURL = server.URL
	config.APIRequestTimeout = 5 * time.Second
	repo := NewRepository()

	// Load the data
	err := repo.LoadData(context.Background())
	if err != nil {
		t.Fatalf("LoadData() failed: %v", err)
	}

	// Check cache status
	if !repo.IsCacheEnabled() {
		t.Error("expected cache to be enabled when WithCache=true")
	}

	// Clean up - remove any cached images
	os.RemoveAll("static/img/artists")

	// Reset cache setting
	config.WithCache = false
}

func TestRepository_InvalidAPIData(t *testing.T) {
	// Create a mock server with malformed relation data
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/artists":
			w.Write([]byte(`[
				{"id": 1, "name": "Test Artist", "creationDate": 2020}
			]`))
		case "/api/relation":
			// Invalid structure - missing "index" wrapper
			w.Write([]byte(`[
				{"id": 1, "datesLocations": {"test-location": ["2023-01-01"]}}
			]`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	config.WithCache = false
	config.APIBaseURL = server.URL
	config.APIRequestTimeout = 5 * time.Second
	repo := NewRepository()

	// Load the data - should fail due to invalid relation structure
	err := repo.LoadData(context.Background())
	if err == nil {
		t.Fatal("expected error for invalid relation structure, got nil")
	}
}

func TestRepository_EdgeCaseData(t *testing.T) {
	// Create a mock server with edge case data
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/artists":
			w.Write([]byte(`[
				{"id": 1, "name": "Artist with Special-Chars! & @#$", "creationDate": 2020, "firstAlbum": "Album"},
				{"id": 2, "name": "", "creationDate": 0, "firstAlbum": ""},
				{"id": 3, "name": "Artist-With-Dashes", "creationDate": 1950, "firstAlbum": "Old Album"}
			]`))
		case "/api/relation":
			w.Write([]byte(`{
				"index": [
					{"id": 1, "datesLocations": {"location-with-special-chars-usa": ["2023-01-01", "2023-01-02"]}},
					{"id": 2, "datesLocations": {}},
					{"id": 3, "datesLocations": {"same-location-usa": ["2020-01-01"], "another-location-uk": ["2021-01-01"]}}
				]
			}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	config.WithCache = false
	config.APIBaseURL = server.URL
	config.APIRequestTimeout = 5 * time.Second
	repo := NewRepository()

	// Load the data
	err := repo.LoadData(context.Background())
	if err != nil {
		t.Fatalf("LoadData() failed: %v", err)
	}

	// Check that all artists are processed correctly
	artists := repo.GetArtists()
	if len(artists) != 3 {
		t.Errorf("expected 3 artists, got %d", len(artists))
	}

	// Check artist with empty name
	emptyNameArtist, found := repo.GetArtistByID(2)
	if !found {
		t.Fatal("could not find artist with ID 2")
	}
	if emptyNameArtist.ConcertCount != 0 {
		t.Errorf("expected artist with empty data to have 0 concerts, got %d", emptyNameArtist.ConcertCount)
	}

	// Check locations are created correctly
	locations := repo.GetLocations()
	if len(locations) < 2 {
		t.Errorf("expected at least 2 locations, got %d", len(locations))
	}

	// Check navigation works with edge cases
	firstPrev, firstNext := repo.GetAdjacentArtists(artists[0].ID)
	lastPrev, lastNext := repo.GetAdjacentArtists(artists[len(artists)-1].ID)

	if firstPrev != nil {
		t.Error("first artist should not have a previous artist")
	}
	if firstNext == nil {
		t.Error("first artist should have a next artist")
	}
	if lastNext != nil {
		t.Error("last artist should not have a next artist")
	}
	if lastPrev == nil {
		t.Error("last artist should have a previous artist")
	}
}
