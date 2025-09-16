// Package app provides tests for the core application functionality.
package repository

import (
	"context"
	"errors"
	"testing"
	"time"
)

// MockAPIResponse simulates the API response for testing
type MockAPIResponse struct {
	artists   []Artist
	concerts  []Concert
	shouldErr bool
}

func (m *MockAPIResponse) FetchAll(ctx context.Context) (*Response, error) {
	if m.shouldErr {
		return nil, errors.New("mock API error")
	}

	return &Response{
		Artists:   m.artists,
		Relations: m.concerts,
	}, nil
}

// Test data that matches Zone01 audit requirements
var testArtists = []Artist{
	{
		ID:           28,
		Name:         "Queen",
		Members:      []string{"Freddie Mercury", "Brian May", "Roger Taylor", "John Deacon", "Mike Grose", "Barry Mitchell", "Doug Fogie"},
		CreationYear: 1970,
		FirstAlbum:   "14-12-1973",
		Image:        "https://groupietrackers.herokuapp.com/api/images/queen.jpeg",
	},
	{
		ID:           14,
		Name:         "Gorillaz",
		Members:      []string{"Damon Albarn", "Jamie Hewlett"},
		CreationYear: 1998,
		FirstAlbum:   "26-03-2001",
		Image:        "https://groupietrackers.herokuapp.com/api/images/gorillaz.jpeg",
	},
	{
		ID:           52,
		Name:         "Travis Scott",
		Members:      []string{"Jacques Berman Webster II"},
		CreationYear: 2008,
		FirstAlbum:   "15-05-2013",
		Image:        "https://groupietrackers.herokuapp.com/api/images/travisscott.jpeg",
	},
}

var testConcerts = []Concert{
	{
		ID: 28, // Queen
		Locations: map[string][]string{
			"london-uk":    {"14-02-1977", "15-02-1977"},
			"paris-france": {"10-03-1979", "11-03-1979"},
			"tokyo-japan":  {"01-05-1985"},
		},
	},
	{
		ID: 14, // Gorillaz
		Locations: map[string][]string{
			"new_york-usa":  {"12-04-2002", "13-04-2002"},
			"manchester-uk": {"20-06-2010"},
		},
	},
	{
		ID: 52, // Travis Scott
		Locations: map[string][]string{
			"houston-usa": {"01-01-2014", "02-01-2014", "03-01-2014"},
			"chicago-usa": {"15-05-2017"},
			"miami-usa":   {"22-09-2018"},
		},
	},
}

func createTestStore() *Repository {
	store := NewRepository("http://test-api", time.Second*5)

	// Manually populate with test data for faster tests
	store.processArtists(testArtists)
	store.processConcerts(testConcerts)
	store.computeStats()

	return store
}

func TestStoreBasicFunctionality(t *testing.T) {
	store := createTestStore()

	// Test artist retrieval
	artists := store.GetArtists()
	if len(artists) != 3 {
		t.Errorf("Expected 3 artists, got %d", len(artists))
	}

	// Test specific artist lookup by ID
	queen, found := store.GetArtist(28)
	if !found {
		t.Error("Expected to find Queen")
	}
	if queen.Name != "Queen" {
		t.Errorf("Expected 'Queen', got '%s'", queen.Name)
	}
	if len(queen.Members) != 7 {
		t.Errorf("Expected Queen to have 7 members, got %d", len(queen.Members))
	}

	// Test artist lookup by slug
	gorillaz, found := store.GetArtistBySlug("gorillaz")
	if !found {
		t.Error("Expected to find Gorillaz by slug")
	}
	if gorillaz.Name != "Gorillaz" {
		t.Errorf("Expected 'Gorillaz', got '%s'", gorillaz.Name)
	}
}

func TestConcertFunctionality(t *testing.T) {
	store := createTestStore()

	// Test concert retrieval
	concert, found := store.GetConcert(28)
	if !found {
		t.Error("Expected to find Queen's concerts")
	}

	// Test show counting
	totalShows := store.CountShows(concert)
	if totalShows != 5 { // 2+2+1 from test data
		t.Errorf("Expected 5 total shows for Queen, got %d", totalShows)
	}

	// Test country extraction
	countries := store.GetCountries(concert)
	if len(countries) != 3 { // uk, france, japan
		t.Errorf("Expected 3 countries for Queen, got %d", len(countries))
	}
}

func TestLocationStats(t *testing.T) {
	store := createTestStore()

	locations := store.GetLocations()
	if len(locations) < 3 {
		t.Errorf("Expected at least 3 unique locations, got %d", len(locations))
	}

	locationStats := store.GetLocationStats()
	if len(locationStats) == 0 {
		t.Error("Expected location stats to be generated")
	}

	// Test that London appears in stats
	londonFound := false
	for _, stat := range locationStats {
		if stat.Name == "london-uk" {
			londonFound = true
			if stat.ArtistCount < 1 {
				t.Error("London should have at least 1 artist")
			}
		}
	}
	if !londonFound {
		t.Error("Expected to find London in location stats")
	}
}

func TestStatistics(t *testing.T) {
	store := createTestStore()

	stats := store.GetStats()

	// Test basic statistics
	if stats["total_artists"] != 3 {
		t.Errorf("Expected 3 total artists, got %d", stats["total_artists"])
	}

	if stats["total_members"] != 10 { // 7+2+1
		t.Errorf("Expected 10 total members, got %d", stats["total_members"])
	}

	if stats["total_shows"] != 13 { // 5+2+3 shows from test data
		t.Errorf("Expected 13 total shows, got %d", stats["total_shows"])
	}
}

func TestNavigationFunctionality(t *testing.T) {
	store := createTestStore()

	// Get sorted artists (alphabetical)
	artists := store.GetArtists()
	if len(artists) < 3 {
		t.Error("Need at least 3 artists for navigation test")
		return
	}

	// Test navigation from middle artist
	middle := artists[1] // Should be middle in sorted order
	prev, next := store.GetNextPrevArtist(middle)

	if prev == nil {
		t.Error("Expected previous artist for middle item")
	}
	if next == nil {
		t.Error("Expected next artist for middle item")
	}
}

func TestAuditComplianceData(t *testing.T) {
	store := createTestStore()

	// Zone01 audit requirements
	queen, found := store.GetArtist(28)
	if !found {
		t.Error("Queen not found")
		return
	}

	if len(queen.Members) != 7 {
		t.Errorf("Queen should have exactly 7 members, got %d", len(queen.Members))
	}

	gorillaz, found := store.GetArtistBySlug("gorillaz")
	if !found {
		t.Error("Gorillaz not found by slug")
		return
	}

	if gorillaz.FirstAlbum != "26-03-2001" {
		t.Errorf("Gorillaz first album should be '26-03-2001', got '%s'", gorillaz.FirstAlbum)
	}

	_, found = store.GetArtist(52)
	if !found {
		t.Error("Travis Scott not found")
		return
	}

	concert, found := store.GetConcert(52)
	if !found {
		t.Error("Travis Scott concerts not found")
		return
	}

	if len(concert.Locations) < 3 {
		t.Errorf("Travis Scott should have at least 3 locations, got %d", len(concert.Locations))
	}
}
