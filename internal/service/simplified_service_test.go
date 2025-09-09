package service

import (
	"testing"

	"groupie-tracker/internal/models"
)

// MockDataStore for testing
type MockDataStore struct {
	artists   []models.Artist
	relations []models.Relation
}

func (m *MockDataStore) GetAllArtists() []models.Artist {
	return m.artists
}

func (m *MockDataStore) GetAllRelations() []models.Relation {
	return m.relations
}

func (m *MockDataStore) SearchArtists(query string) []models.Artist {
	// Simple implementation for testing
	if query == "" {
		return m.artists
	}

	var results []models.Artist
	for _, artist := range m.artists {
		if artist.Name == query {
			results = append(results, artist)
		}
	}
	return results
}

func TestNewSimplifiedService(t *testing.T) {
	mockStore := &MockDataStore{}
	service := NewSimplifiedService(mockStore)

	if service == nil {
		t.Fatal("NewSimplifiedService() returned nil")
	}

	if service.store != mockStore {
		t.Error("Service store not set correctly")
	}
}

func TestSimplifiedServiceCalculateLocationStats(t *testing.T) {
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

	service := NewSimplifiedService(mockStore)
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

func TestSimplifiedServiceSortLocationStatsByConcertCount(t *testing.T) {
	mockStore := &MockDataStore{}
	service := NewSimplifiedService(mockStore)

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

func TestSimplifiedServiceCalculateTotalCountries(t *testing.T) {
	mockStore := &MockDataStore{}
	service := NewSimplifiedService(mockStore)

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

func TestSimplifiedServiceCalculateTotalConcerts(t *testing.T) {
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

	service := NewSimplifiedService(mockStore)
	totalConcerts := service.CalculateTotalConcerts()

	// Should be 6 total concerts (2 + 1 + 3)
	if totalConcerts != 6 {
		t.Errorf("Expected 6 total concerts, got %d", totalConcerts)
	}
}

func TestSimplifiedServiceGetMostPopularLocations(t *testing.T) {
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

	service := NewSimplifiedService(mockStore)
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
