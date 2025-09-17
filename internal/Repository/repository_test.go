// Package repository provides tests for the core application functionality.
package repository

import (
	"testing"
	"time"
)

// Test data that matches Zone01 audit requirements
var testArtists = []Artist{
	{
		ID:           28,
		Name:         "Queen",
		Members:      []string{"Freddie Mercury", "Brian May", "Roger Taylor", "John Deacon", "Mike Grose", "Barry Mitchell", "Doug Fogie"},
		CreationYear: 1970,
		FirstAlbum:   "14-12-1973",
		Image:        "https://groupietrackers.herokuapp.com/api/images/queen.jpeg",
		Concerts: map[string][]string{
			"london-uk":    {"14-02-1977", "15-02-1977"},
			"paris-france": {"10-03-1979", "11-03-1979"},
			"tokyo-japan":  {"01-05-1985"},
		},
	},
	{
		ID:           14,
		Name:         "Gorillaz",
		Members:      []string{"Damon Albarn", "Jamie Hewlett"},
		CreationYear: 1998,
		FirstAlbum:   "26-03-2001",
		Image:        "https://groupietrackers.herokuapp.com/api/images/gorillaz.jpeg",
		Concerts: map[string][]string{
			"new_york-usa":  {"12-04-2002", "13-04-2002"},
			"manchester-uk": {"20-06-2010"},
		},
	},
	{
		ID:           52,
		Name:         "Travis Scott",
		Members:      []string{"Jacques Berman Webster II"},
		CreationYear: 2008,
		FirstAlbum:   "15-05-2013",
		Image:        "https://groupietrackers.herokuapp.com/api/images/travisscott.jpeg",
		Concerts: map[string][]string{
			"houston-usa": {"01-01-2014", "02-01-2014", "03-01-2014"},
			"chicago-usa": {"15-05-2017"},
			"miami-usa":   {"22-09-2018"},
		},
	},
}

var testRelations = []struct {
	ID             int                 `json:"id"`
	DatesLocations map[string][]string `json:"datesLocations"`
}{
	{
		ID: 28,
		DatesLocations: map[string][]string{
			"london-uk":    {"14-02-1977", "15-02-1977"},
			"paris-france": {"10-03-1979", "11-03-1979"},
			"tokyo-japan":  {"01-05-1985"},
		},
	},
	{
		ID: 14,
		DatesLocations: map[string][]string{
			"new_york-usa":  {"12-04-2002", "13-04-2002"},
			"manchester-uk": {"20-06-2010"},
		},
	},
	{
		ID: 52,
		DatesLocations: map[string][]string{
			"houston-usa": {"01-01-2014", "02-01-2014", "03-01-2014"},
			"chicago-usa": {"15-05-2017"},
			"miami-usa":   {"22-09-2018"},
		},
	},
}

func createTestRepository() *Repository {
	repo := NewRepository("http://test-api", time.Second*5)

	// Manually populate with test data
	repo.processArtists(testArtists)
	repo.processRelations(testRelations)
	repo.computeLocationStats()

	return repo
}

func TestRepositoryBasicFunctionality(t *testing.T) {
	repo := createTestRepository()

	// Test artist retrieval
	artists := repo.GetArtists()
	if len(artists) != 3 {
		t.Errorf("Expected 3 artists, got %d", len(artists))
	}

	// Test specific artist lookup by ID
	queen, found := repo.GetArtist(28)
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
	gorillaz, found := repo.GetArtistBySlug("gorillaz")
	if !found {
		t.Error("Expected to find Gorillaz by slug")
	}
	if gorillaz.Name != "Gorillaz" {
		t.Errorf("Expected 'Gorillaz', got '%s'", gorillaz.Name)
	}
}

func TestArtistConcerts(t *testing.T) {
	repo := createTestRepository()

	// Test concert data
	queen, found := repo.GetArtist(28)
	if !found {
		t.Error("Expected to find Queen")
		return
	}

	// Test concert counting
	totalConcerts := repo.CountConcerts(queen)
	if totalConcerts != 5 { // 2+2+1 from test data
		t.Errorf("Expected 5 total concerts for Queen, got %d", totalConcerts)
	}

	// Test country extraction
	countries := repo.GetCountries(queen)
	if len(countries) != 3 { // uk, france, japan
		t.Errorf("Expected 3 countries for Queen, got %d", len(countries))
	}
}

func TestLocationStats(t *testing.T) {
	repo := createTestRepository()

	locations := repo.GetLocations()
	if len(locations) < 3 {
		t.Errorf("Expected at least 3 unique locations, got %d", len(locations))
	}

	locationStats := repo.GetLocationStats()
	if len(locationStats) == 0 {
		t.Error("Expected location stats to be generated")
	}

	// Test that London appears in stats
	londonFound := false
	for _, stat := range locationStats {
		if stat.Name == "london-uk" {
			londonFound = true
			if len(stat.Artists) < 1 {
				t.Error("London should have at least 1 artist")
			}
		}
	}
	if !londonFound {
		t.Error("Expected to find London in location stats")
	}
}

func TestStatistics(t *testing.T) {
	repo := createTestRepository()

	stats := repo.GetStats()

	// Test basic statistics
	if stats["total_artists"] != 3 {
		t.Errorf("Expected 3 total artists, got %d", stats["total_artists"])
	}

	if stats["total_members"] != 10 { // 7+2+1
		t.Errorf("Expected 10 total members, got %d", stats["total_members"])
	}

	if stats["total_concerts"] != 13 { // 5+2+6 concerts from test data (Queen: 5, Gorillaz: 2, Travis: 6)
		t.Errorf("Expected 13 total concerts, got %d", stats["total_concerts"])
	}
}

func TestNavigationFunctionality(t *testing.T) {
	repo := createTestRepository()

	// Get sorted artists (alphabetical)
	artists := repo.GetArtists()
	if len(artists) < 3 {
		t.Error("Need at least 3 artists for navigation test")
		return
	}

	// Test navigation from middle artist
	middle := artists[1] // Should be middle in sorted order
	prev, next := repo.GetNextPrevArtist(middle)

	if prev == nil {
		t.Error("Expected previous artist for middle item")
	}
	if next == nil {
		t.Error("Expected next artist for middle item")
	}
}

func TestAuditComplianceData(t *testing.T) {
	repo := createTestRepository()

	// Zone01 audit requirements
	queen, found := repo.GetArtist(28)
	if !found {
		t.Error("Queen not found")
		return
	}

	if len(queen.Members) != 7 {
		t.Errorf("Queen should have exactly 7 members, got %d", len(queen.Members))
	}

	gorillaz, found := repo.GetArtistBySlug("gorillaz")
	if !found {
		t.Error("Gorillaz not found by slug")
		return
	}

	if gorillaz.FirstAlbum != "26-03-2001" {
		t.Errorf("Gorillaz first album should be '26-03-2001', got '%s'", gorillaz.FirstAlbum)
	}

	travisScott, found := repo.GetArtist(52)
	if !found {
		t.Error("Travis Scott not found")
		return
	}

	if len(travisScott.Concerts) < 3 {
		t.Errorf("Travis Scott should have at least 3 locations, got %d", len(travisScott.Concerts))
	}
}

func TestLocationBySlug(t *testing.T) {
	repo := createTestRepository()

	// Test location retrieval by slug
	location, found := repo.GetLocationBySlug("london-uk")
	if !found {
		t.Error("Expected to find London by slug")
	}

	if location.Name != "london-uk" {
		t.Errorf("Expected location name 'london-uk', got '%s'", location.Name)
	}

	if len(location.Artists) == 0 {
		t.Error("Expected London to have at least one artist")
	}
}

func TestSlugGeneration(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Queen", "queen"},
		{"Foo Fighters", "foo-fighters"},
		{"Green Day", "green-day"},
		{"AC/DC", "ac-dc"},
		{"Twenty One Pilots", "twenty-one-pilots"},
	}

	for _, test := range tests {
		result := createSlug(test.input)
		if result != test.expected {
			t.Errorf("createSlug(%s) = %s, expected %s", test.input, result, test.expected)
		}
	}
}

func TestNewRepository(t *testing.T) {
	repo := NewRepository("http://test-api", time.Second*10)
	if repo == nil {
		t.Fatal("NewRepository should not return nil")
	}

	if repo.baseURL != "http://test-api" {
		t.Errorf("Expected baseURL 'http://test-api', got '%s'", repo.baseURL)
	}

	if repo.client == nil {
		t.Fatal("Repository client should not be nil")
	}

	if repo.client.Timeout != time.Second*10 {
		t.Errorf("Expected timeout 10s, got %v", repo.client.Timeout)
	}
}

func TestGetArtistNotFound(t *testing.T) {
	repo := createTestRepository()

	// Test getting non-existent artist
	_, found := repo.GetArtist(999)
	if found {
		t.Error("Should not find non-existent artist")
	}
}

func TestGetArtistBySlugNotFound(t *testing.T) {
	repo := createTestRepository()

	// Test getting non-existent artist by slug
	_, found := repo.GetArtistBySlug("nonexistent")
	if found {
		t.Error("Should not find non-existent artist by slug")
	}
}

func TestGetLocationBySlugNotFound(t *testing.T) {
	repo := createTestRepository()

	// Test getting non-existent location
	_, found := repo.GetLocationBySlug("nonexistent")
	if found {
		t.Error("Should not find non-existent location")
	}
}

func TestContainsArtistPtr(t *testing.T) {
	artist1 := &Artist{ID: 1, Name: "Test1"}
	artist2 := &Artist{ID: 2, Name: "Test2"}

	artists := []Artist{*artist1}

	// Test finding existing artist
	if !containsArtistPtr(artists, artist1) {
		t.Error("Should find existing artist")
	}

	// Test not finding non-existing artist
	if containsArtistPtr(artists, artist2) {
		t.Error("Should not find non-existing artist")
	}
}

func TestNormalizeLocation(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"London-UK", "london-uk"},
		{"NEW YORK-USA", "new york-usa"},
		{"  Paris-France  ", "paris-france"},
		{"", ""},
	}

	for _, test := range tests {
		result := normalizeLocation(test.input)
		if result != test.expected {
			t.Errorf("normalizeLocation(%q) = %q, expected %q", test.input, result, test.expected)
		}
	}
}
