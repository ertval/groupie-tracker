package service

import (
	"testing"

	"groupie-tracker/internal/models"
)

// MockDataStore for testing
type MockDataStore struct {
	artists   []models.Artist
	relations []models.Relation
	locations []string
	dates     []string
}

func (m *MockDataStore) GetAllArtists() []models.Artist {
	return m.artists
}

func (m *MockDataStore) GetAllRelations() []models.Relation {
	return m.relations
}

func (m *MockDataStore) GetUniqueLocations() []string {
	return m.locations
}

func (m *MockDataStore) GetUniqueDates() []string {
	return m.dates
}

func TestNewService(t *testing.T) {
	mockStore := &MockDataStore{}
	service := NewService(mockStore)

	if service == nil {
		t.Fatal("NewService() returned nil")
	}

	if service.store != mockStore {
		t.Error("Service store not set correctly")
	}
}

func TestServiceCalculateLocationStats(t *testing.T) {
	// Create test data
	artists := []models.Artist{
		{ID: 1, Name: "Queen"},
		{ID: 2, Name: "AC/DC"},
	}

	relations := []models.Relation{
		{
			ID: 1,
			DatesLocations: map[string][]string{
				"london-uk":    {"01-01-1980", "02-01-1980", "03-01-1980"}, // 3 concerts
				"new_york-usa": {"04-01-1980"},                             // 1 concert
			},
		},
		{
			ID: 2,
			DatesLocations: map[string][]string{
				"london-uk":        {"01-02-1981", "02-02-1981"}, // 2 more concerts (5 total)
				"sydney-australia": {"03-02-1981"},               // 1 concert
			},
		},
	}

	mockStore := &MockDataStore{
		artists:   artists,
		relations: relations,
	}

	service := NewService(mockStore)
	stats := service.CalculateLocationStats()

	// Verify we have the expected number of locations
	if len(stats) != 3 {
		t.Errorf("Expected 3 location stats, got %d", len(stats))
	}

	// Create a map for easier verification
	statsMap := make(map[string]LocationStat)
	for _, stat := range stats {
		statsMap[stat.Name] = stat
	}

	// Verify london-uk has the most concerts
	londonStat, exists := statsMap["london-uk"]
	if !exists {
		t.Error("london-uk not found in stats")
	} else {
		if londonStat.ConcertCount != 5 {
			t.Errorf("Expected 5 concerts for london-uk, got %d", londonStat.ConcertCount)
		}
		if londonStat.ArtistCount != 2 {
			t.Errorf("Expected 2 artists for london-uk, got %d", londonStat.ArtistCount)
		}
	}

	// Verify new_york-usa
	nyStats, exists := statsMap["new_york-usa"]
	if !exists {
		t.Error("new_york-usa not found in stats")
	} else {
		if nyStats.ConcertCount != 1 {
			t.Errorf("Expected 1 concert for new_york-usa, got %d", nyStats.ConcertCount)
		}
		if nyStats.ArtistCount != 1 {
			t.Errorf("Expected 1 artist for new_york-usa, got %d", nyStats.ArtistCount)
		}
	}
}

func TestServiceSortLocationStatsByConcertCount(t *testing.T) {
	mockStore := &MockDataStore{}
	service := NewService(mockStore)

	// Create test location stats
	stats := []LocationStat{
		{Name: "location1", ConcertCount: 3, ArtistCount: 2},
		{Name: "location2", ConcertCount: 7, ArtistCount: 1},
		{Name: "location3", ConcertCount: 1, ArtistCount: 3},
		{Name: "location4", ConcertCount: 5, ArtistCount: 2},
	}

	// Sort by concert count
	sortedStats := service.SortLocationStatsByConcertCount(stats)

	// Verify sorting order (descending by concert count)
	expectedOrder := []string{"location2", "location4", "location1", "location3"}
	expectedCounts := []int{7, 5, 3, 1}

	if len(sortedStats) != len(expectedOrder) {
		t.Errorf("Expected %d locations, got %d", len(expectedOrder), len(sortedStats))
	}

	for i, expectedName := range expectedOrder {
		if sortedStats[i].Name != expectedName {
			t.Errorf("Position %d: expected %s, got %s", i, expectedName, sortedStats[i].Name)
		}
		if sortedStats[i].ConcertCount != expectedCounts[i] {
			t.Errorf("Position %d: expected %d concerts, got %d", i, expectedCounts[i], sortedStats[i].ConcertCount)
		}
	}
}

func TestServiceCalculateTotalCountries(t *testing.T) {
	mockStore := &MockDataStore{}
	service := NewService(mockStore)

	// Create test location stats with different countries
	stats := []LocationStat{
		{Name: "london-uk"},
		{Name: "manchester-uk"}, // Same country
		{Name: "new_york-usa"},
		{Name: "los_angeles-usa"}, // Same country
		{Name: "sydney-australia"},
		{Name: "tokyo-japan"},
	}

	totalCountries := service.CalculateTotalCountries(stats)

	// Should be 4 unique countries: uk, usa, australia, japan
	if totalCountries != 4 {
		t.Errorf("Expected 4 unique countries, got %d", totalCountries)
	}
}

func TestServiceCalculateTotalConcerts(t *testing.T) {
	relations := []models.Relation{
		{
			ID: 1,
			DatesLocations: map[string][]string{
				"london-uk":    {"01-01-1980", "02-01-1980"}, // 2 concerts
				"new_york-usa": {"03-01-1980"},               // 1 concert
			},
		},
		{
			ID: 2,
			DatesLocations: map[string][]string{
				"sydney-australia": {"01-02-1981", "02-02-1981", "03-02-1981"}, // 3 concerts
			},
		},
	}

	mockStore := &MockDataStore{
		relations: relations,
	}

	service := NewService(mockStore)
	totalConcerts := service.CalculateTotalConcerts()

	// Should be 6 total concerts (2 + 1 + 3)
	if totalConcerts != 6 {
		t.Errorf("Expected 6 total concerts, got %d", totalConcerts)
	}
}

func TestServiceGetMostPopularLocations(t *testing.T) {
	relations := []models.Relation{
		{
			ID: 1,
			DatesLocations: map[string][]string{
				"london-uk":    {"01-01-1980", "02-01-1980", "03-01-1980"}, // 3 concerts
				"new_york-usa": {"04-01-1980"},                             // 1 concert
			},
		},
		{
			ID: 2,
			DatesLocations: map[string][]string{
				"london-uk":        {"01-02-1981", "02-02-1981"}, // 2 more concerts (5 total)
				"sydney-australia": {"03-02-1981"},               // 1 concert
				"new_york-usa":     {"04-02-1981"},               // 1 more concert (2 total)
			},
		},
	}

	mockStore := &MockDataStore{
		relations: relations,
	}

	service := NewService(mockStore)
	popularLocations := service.GetMostPopularLocations(3)

	// Should return top 3 locations by concert count
	if len(popularLocations) != 3 {
		t.Errorf("Expected 3 popular locations, got %d", len(popularLocations))
	}

	// Verify order (should be sorted by concert count descending)
	expected := []struct {
		location string
		count    int
	}{
		{"london-uk", 5},
		{"new_york-usa", 2},
		{"sydney-australia", 1},
	}

	for i, exp := range expected {
		if popularLocations[i].Location != exp.location {
			t.Errorf("Position %d: expected location %s, got %s", i, exp.location, popularLocations[i].Location)
		}
		if popularLocations[i].Count != exp.count {
			t.Errorf("Position %d: expected count %d, got %d", i, exp.count, popularLocations[i].Count)
		}
	}
}

func TestServiceSearchArtists(t *testing.T) {
	// Create test data
	artists := []models.Artist{
		{ID: 1, Name: "Queen", Members: []string{"Freddie Mercury", "Brian May"}},
		{ID: 2, Name: "Beatles", Members: []string{"John Lennon", "Paul McCartney"}},
		{ID: 3, Name: "Led Zeppelin", Members: []string{"Robert Plant", "Jimmy Page"}},
	}

	mockStore := &MockDataStore{
		artists: artists,
	}

	service := NewService(mockStore)

	// Test search by artist name
	results := service.SearchArtists("Queen")
	if len(results) != 1 {
		t.Errorf("Expected 1 result for 'Queen', got %d", len(results))
	}
	if results[0].Name != "Queen" {
		t.Errorf("Expected 'Queen', got %s", results[0].Name)
	}

	// Test search by member name
	results = service.SearchArtists("Freddie")
	if len(results) != 1 {
		t.Errorf("Expected 1 result for 'Freddie', got %d", len(results))
	}
	if results[0].Name != "Queen" {
		t.Errorf("Expected 'Queen', got %s", results[0].Name)
	}

	// Test case-insensitive search
	results = service.SearchArtists("beatles")
	if len(results) != 1 {
		t.Errorf("Expected 1 result for 'beatles', got %d", len(results))
	}
	if results[0].Name != "Beatles" {
		t.Errorf("Expected 'Beatles', got %s", results[0].Name)
	}

	// Test empty search returns all artists sorted
	results = service.SearchArtists("")
	if len(results) != 3 {
		t.Errorf("Expected 3 results for empty search, got %d", len(results))
	}

	// Verify results are sorted alphabetically
	expectedOrder := []string{"Beatles", "Led Zeppelin", "Queen"}
	for i, artist := range results {
		if artist.Name != expectedOrder[i] {
			t.Errorf("Position %d: expected %s, got %s", i, expectedOrder[i], artist.Name)
		}
	}
}

func TestServiceFilterArtistsByYear(t *testing.T) {
	// Create test data with different creation years
	artists := []models.Artist{
		{ID: 1, Name: "Old Band", CreationYear: 1960},
		{ID: 2, Name: "Medium Band", CreationYear: 1980},
		{ID: 3, Name: "New Band", CreationYear: 2000},
	}

	mockStore := &MockDataStore{
		artists: artists,
	}

	service := NewService(mockStore)

	// Test filter by minimum year
	results := service.FilterArtistsByYear(1970, 0)
	if len(results) != 2 {
		t.Errorf("Expected 2 results for min year 1970, got %d", len(results))
	}

	// Test filter by maximum year
	results = service.FilterArtistsByYear(0, 1990)
	if len(results) != 2 {
		t.Errorf("Expected 2 results for max year 1990, got %d", len(results))
	}

	// Test filter by year range
	results = service.FilterArtistsByYear(1970, 1990)
	if len(results) != 1 {
		t.Errorf("Expected 1 result for year range 1970-1990, got %d", len(results))
	}
	if results[0].Name != "Medium Band" {
		t.Errorf("Expected 'Medium Band', got %s", results[0].Name)
	}

	// Test no filter (all years)
	results = service.FilterArtistsByYear(0, 0)
	if len(results) != 3 {
		t.Errorf("Expected 3 results for no filter, got %d", len(results))
	}
}

func TestServiceGetStats(t *testing.T) {
	// Create test data
	artists := []models.Artist{
		{ID: 1, Name: "Queen"},
		{ID: 2, Name: "AC/DC"},
	}

	relations := []models.Relation{
		{
			ID: 1,
			DatesLocations: map[string][]string{
				"london-uk":    {"01-01-1980", "02-01-1980", "03-01-1980"}, // 3 concerts
				"new_york-usa": {"04-01-1980"},                             // 1 concert
			},
		},
		{
			ID: 2,
			DatesLocations: map[string][]string{
				"london-uk":        {"01-02-1981", "02-02-1981"}, // 2 more concerts
				"sydney-australia": {"03-02-1981"},               // 1 concert
			},
		},
	}

	locations := []string{"london-uk", "new_york-usa", "sydney-australia"}
	dates := []string{"01-01-1980", "02-01-1980", "03-01-1980", "04-01-1980", "01-02-1981", "02-02-1981", "03-02-1981"}

	mockStore := &MockDataStore{
		artists:   artists,
		relations: relations,
		locations: locations,
		dates:     dates,
	}

	service := NewService(mockStore)
	stats := service.GetStats()

	// Check basic counts
	if stats["artists"] != 2 {
		t.Errorf("Expected 2 artists, got %d", stats["artists"])
	}
	if stats["relations"] != 2 {
		t.Errorf("Expected 2 relations, got %d", stats["relations"])
	}
	if stats["locations"] != 3 {
		t.Errorf("Expected 3 locations, got %d", stats["locations"])
	}
	if stats["dates"] != 7 {
		t.Errorf("Expected 7 dates, got %d", stats["dates"])
	}

	// Check total concerts calculation
	if stats["total_concerts"] != 7 {
		t.Errorf("Expected 7 total concerts, got %d", stats["total_concerts"])
	}
}

// TestServiceCalculateTotalShows tests the CalculateTotalShows function
func TestServiceCalculateTotalShows(t *testing.T) {
	service := NewService(&MockDataStore{})

	// Create test relation
	relation := models.Relation{
		ID: 1,
		DatesLocations: map[string][]string{
			"london-uk":    {"01-01-2020", "02-01-2020"},
			"new_york-usa": {"03-01-2020"},
			"paris-france": {"04-01-2020", "05-01-2020", "06-01-2020"},
		},
	}

	totalShows := service.CalculateTotalShows(relation)

	// Expected: 2 + 1 + 3 = 6 shows
	expected := 6
	if totalShows != expected {
		t.Errorf("Expected %d total shows, got %d", expected, totalShows)
	}
}

// TestServiceExtractCountries tests the ExtractCountries function
func TestServiceExtractCountries(t *testing.T) {
	service := NewService(&MockDataStore{})

	// Create test relation with multiple countries
	relation := models.Relation{
		ID: 1,
		DatesLocations: map[string][]string{
			"london-uk":       {"01-01-2020"},
			"manchester-uk":   {"02-01-2020"},
			"new_york-usa":    {"03-01-2020"},
			"los_angeles-usa": {"04-01-2020"},
			"paris-france":    {"05-01-2020"},
			"berlin-germany":  {"06-01-2020"},
		},
	}

	countries := service.ExtractCountries(relation)

	// Should extract: uk, usa, france, germany = 4 unique countries
	expectedCount := 4
	if len(countries) != expectedCount {
		t.Errorf("Expected %d countries, got %d", expectedCount, len(countries))
	}

	// Verify specific countries are present
	expectedCountries := map[string]bool{
		"uk":      true,
		"usa":     true,
		"france":  true,
		"germany": true,
	}

	for _, country := range countries {
		if !expectedCountries[country] {
			t.Errorf("Unexpected country: %s", country)
		}
	}
}
