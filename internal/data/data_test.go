package data

import (
	"context"
	"testing"

	"groupie-tracker/internal/client"
)

func TestNewRepository(t *testing.T) {
	repo := NewRepository()
	if repo == nil {
		t.Error("NewRepository() returned nil")
	}
	// Note: Fields may be initialized as empty slices rather than nil
}

func getTestData() *client.Response {
	return &client.Response{
		Artists: []client.Artist{
			{
				ID:           1,
				Name:         "Queen",
				Members:      []string{"Freddie Mercury", "Brian May", "John Deacon", "Roger Taylor"},
				CreationYear: 1970,
				FirstAlbum:   "14-07-1973",
				Image:        "https://groupietrackers.herokuapp.com/api/images/queen.jpeg",
			},
			{
				ID:           2,
				Name:         "Pink Floyd",
				Members:      []string{"David Gilmour", "Roger Waters", "Richard Wright", "Nick Mason"},
				CreationYear: 1965,
				FirstAlbum:   "05-08-1967",
				Image:        "https://groupietrackers.herokuapp.com/api/images/pink_floyd.jpeg",
			},
		},
		Relations: []client.Relation{
			{
				ID: 1,
				DatesLocations: map[string][]string{
					"new_york-usa": {"23-08-2019", "22-08-2020"},
					"london-uk":    {"20-02-2019", "25-07-2020"},
					"paris-france": {"28-09-2019"},
				},
			},
			{
				ID: 2,
				DatesLocations: map[string][]string{
					"tokyo-japan":      {"15-03-2019", "10-05-2020"},
					"berlin-germany":   {"18-04-2019"},
					"sydney-australia": {"22-11-2019", "05-12-2020"},
				},
			},
		},
	}
}

func TestInitializeWithData(t *testing.T) {
	repo := NewRepository()
	testData := getTestData()

	ctx := context.Background()
	err := repo.InitializeWithData(ctx, testData)
	if err != nil {
		t.Errorf("InitializeWithData failed: %v", err)
	}

	// Verify artists were loaded
	artists := repo.GetAllArtists()
	if len(artists) != 2 {
		t.Errorf("Expected 2 artists, got %d", len(artists))
	}

	// Verify specific artist data
	queen, found := repo.GetArtistBySlug("queen")
	if !found {
		t.Error("Expected to find Queen by slug")
	}
	if queen.Name != "Queen" {
		t.Errorf("Expected name 'Queen', got '%s'", queen.Name)
	}
	if len(queen.Members) != 4 {
		t.Errorf("Expected 4 members for Queen, got %d", len(queen.Members))
	}
}

func TestGetAllArtists(t *testing.T) {
	repo := NewRepository()
	testData := getTestData()
	repo.InitializeWithData(context.Background(), testData)

	artists := repo.GetAllArtists()
	if len(artists) != 2 {
		t.Errorf("Expected 2 artists, got %d", len(artists))
	}
}

func TestGetAllArtistsSorted(t *testing.T) {
	repo := NewRepository()
	testData := getTestData()
	repo.InitializeWithData(context.Background(), testData)

	artists := repo.GetAllArtistsSorted()
	if len(artists) != 2 {
		t.Errorf("Expected 2 artists, got %d", len(artists))
	}

	// Check if sorted alphabetically (Pink Floyd should come before Queen)
	if artists[0].Name != "Pink Floyd" {
		t.Errorf("Expected first artist to be 'Pink Floyd', got '%s'", artists[0].Name)
	}
	if artists[1].Name != "Queen" {
		t.Errorf("Expected second artist to be 'Queen', got '%s'", artists[1].Name)
	}
}

func TestGetArtist(t *testing.T) {
	repo := NewRepository()
	testData := getTestData()
	repo.InitializeWithData(context.Background(), testData)

	// Test existing artist
	artist, found := repo.GetArtist(1)
	if !found {
		t.Error("Expected to find artist with ID 1")
	}
	if artist.Name != "Queen" {
		t.Errorf("Expected 'Queen', got '%s'", artist.Name)
	}

	// Test non-existing artist
	_, found = repo.GetArtist(999)
	if found {
		t.Error("Expected not to find artist with ID 999")
	}
}

func TestGetArtistBySlug(t *testing.T) {
	repo := NewRepository()
	testData := getTestData()
	repo.InitializeWithData(context.Background(), testData)

	// Test existing artist
	artist, found := repo.GetArtistBySlug("queen")
	if !found {
		t.Error("Expected to find artist with slug 'queen'")
	}
	if artist.Name != "Queen" {
		t.Errorf("Expected 'Queen', got '%s'", artist.Name)
	}

	// Test non-existing artist
	_, found = repo.GetArtistBySlug("nonexistent")
	if found {
		t.Error("Expected not to find artist with slug 'nonexistent'")
	}
}

func TestGetRelation(t *testing.T) {
	repo := NewRepository()
	testData := getTestData()
	repo.InitializeWithData(context.Background(), testData)

	// Test existing relation
	relation, found := repo.GetRelation(1)
	if !found {
		t.Error("Expected to find relation with ID 1")
	}
	if len(relation.DatesLocations) == 0 {
		t.Error("Expected relation to have dates and locations")
	}

	// Test non-existing relation
	_, found = repo.GetRelation(999)
	if found {
		t.Error("Expected not to find relation with ID 999")
	}
}

func TestGetStats(t *testing.T) {
	repo := NewRepository()
	testData := getTestData()
	repo.InitializeWithData(context.Background(), testData)

	stats := repo.GetStats()
	if stats["artists"] != 2 {
		t.Errorf("Expected 2 artists in stats, got %d", stats["artists"])
	}
	if stats["relations"] != 2 {
		t.Errorf("Expected 2 relations in stats, got %d", stats["relations"])
	}
}

func TestGetTotalMembers(t *testing.T) {
	repo := NewRepository()
	testData := getTestData()
	repo.InitializeWithData(context.Background(), testData)

	total := repo.GetTotalMembers()
	if total != 8 { // 4 Queen members + 4 Pink Floyd members
		t.Errorf("Expected 8 total members, got %d", total)
	}
}

func TestGetTotalCountries(t *testing.T) {
	repo := NewRepository()
	testData := getTestData()
	repo.InitializeWithData(context.Background(), testData)

	total := repo.GetTotalCountries()
	if total == 0 {
		t.Error("Expected positive number of countries")
	}
}

func TestGetUniqueLocations(t *testing.T) {
	repo := NewRepository()
	testData := getTestData()
	repo.InitializeWithData(context.Background(), testData)

	locations := repo.GetUniqueLocations()
	if len(locations) == 0 {
		t.Error("Expected some unique locations")
	}
}

func TestCalculateLocationStats(t *testing.T) {
	repo := NewRepository()
	testData := getTestData()
	repo.InitializeWithData(context.Background(), testData)

	stats := repo.CalculateLocationStats()
	if len(stats) == 0 {
		t.Error("Expected some location stats")
	}
}

func TestGetArtistNavigation(t *testing.T) {
	repo := NewRepository()
	testData := getTestData()
	repo.InitializeWithData(context.Background(), testData)

	// Get first artist for navigation test
	artists := repo.GetAllArtistsSorted()
	if len(artists) < 2 {
		t.Skip("Need at least 2 artists for navigation test")
	}

	prev, next := repo.GetArtistNavigation(artists[0])
	if prev != nil {
		t.Error("Expected no previous artist for first artist")
	}
	if next == nil {
		t.Error("Expected next artist for first artist")
	}
}

func TestCalculateTotalShows(t *testing.T) {
	repo := NewRepository()
	testData := getTestData()
	repo.InitializeWithData(context.Background(), testData)

	relation, found := repo.GetRelation(1)
	if !found {
		t.Skip("Need relation data for this test")
	}

	total := repo.CalculateTotalShows(relation)
	if total == 0 {
		t.Error("Expected some shows for Queen")
	}
}

func TestExtractCountries(t *testing.T) {
	repo := NewRepository()
	testData := getTestData()
	repo.InitializeWithData(context.Background(), testData)

	relation, found := repo.GetRelation(1)
	if !found {
		t.Skip("Need relation data for this test")
	}

	countries := repo.ExtractCountries(relation)
	if len(countries) == 0 {
		t.Error("Expected some countries")
	}

	// Should include USA, UK, FRANCE from the test data (in uppercase)
	expectedCountries := []string{"USA", "UK", "FRANCE"}
	for _, expected := range expectedCountries {
		found := false
		for _, country := range countries {
			if country == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected to find country '%s' in extracted countries", expected)
		}
	}
}

func TestSlugGeneration(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Queen", "queen"},
		{"Pink Floyd", "pink-floyd"},
		{"AC/DC", "ac-dc"},
		{"The Beatles", "the-beatles"},
	}

	for _, test := range tests {
		result := generateSlug(test.input)
		if result != test.expected {
			t.Errorf("generateSlug(%s) = %s, expected %s", test.input, result, test.expected)
		}
	}
}

func TestGenerateLocationSlug(t *testing.T) {
	result := GenerateLocationSlug("new_york-usa")
	expected := "new-york-usa"
	if result != expected {
		t.Errorf("GenerateLocationSlug('new_york-usa') = %s, expected %s", result, expected)
	}
}

func TestNormalizeLocationName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"new_york-usa", "New York, USA"},
		{"london-uk", "London, UK"},
		{"paris-france", "Paris, FRANCE"},
		{"los_angeles-usa", "Los Angeles, USA"},
	}

	for _, test := range tests {
		result := NormalizeLocationName(test.input)
		if result != test.expected {
			t.Errorf("NormalizeLocationName(%s) = %s, expected %s", test.input, result, test.expected)
		}
	}
}

func TestGetLocationDetailsBySlug(t *testing.T) {
	repo := NewRepository()
	testData := getTestData()
	repo.InitializeWithData(context.Background(), testData)

	// Test with existing location
	details, found := repo.GetLocationDetailsBySlug("new-york-usa")
	if !found {
		t.Error("Expected to find location details for 'new-york-usa'")
	}
	// Check that we got some valid data (the actual name format may vary)
	if details.Name == "" {
		t.Error("Expected location details to have a name")
	}

	// Test with non-existing location
	_, found = repo.GetLocationDetailsBySlug("nonexistent")
	if found {
		t.Error("Expected not to find location details for 'nonexistent'")
	}
}

func TestGetArtistsWithDatesForLocation(t *testing.T) {
	repo := NewRepository()
	testData := getTestData()
	repo.InitializeWithData(context.Background(), testData)

	artists := repo.GetArtistsWithDatesForLocation("new_york-usa")
	if len(artists) == 0 {
		t.Error("Expected to find artists for new_york-usa location")
	}

	// Check first artist data
	if len(artists) > 0 {
		artist := artists[0]
		if artist.Artist.Name == "" {
			t.Error("Expected artist name to be set")
		}
		if len(artist.Dates) == 0 {
			t.Error("Expected some dates for the artist")
		}
	}
}
