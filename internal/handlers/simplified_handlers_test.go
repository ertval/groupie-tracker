package handlers

import (
	"testing"

	"groupie-tracker/internal/models"
	"groupie-tracker/internal/service"
)

// SimplifiedMockStore for testing the new architecture
type SimplifiedMockStore struct {
	artists   []models.Artist
	relations []models.Relation
}

func (m *SimplifiedMockStore) GetAllArtists() []models.Artist {
	return m.artists
}

func (m *SimplifiedMockStore) GetAllRelations() []models.Relation {
	return m.relations
}

func (m *SimplifiedMockStore) SearchArtists(query string) []models.Artist {
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

func (m *SimplifiedMockStore) GetStats() map[string]int {
	return map[string]int{
		"artists":   len(m.artists),
		"relations": len(m.relations),
	}
}

func (m *SimplifiedMockStore) GetUniqueLocations() []string {
	locationSet := make(map[string]bool)
	for _, relation := range m.relations {
		for location := range relation.DatesLocations {
			locationSet[location] = true
		}
	}

	var locations []string
	for location := range locationSet {
		locations = append(locations, location)
	}
	return locations
}

func TestSimplifiedHandlersWithSortedLocations(t *testing.T) {
	// Create test data where we can verify sorting
	artists := []models.Artist{
		{ID: 1, Name: "Artist1"},
		{ID: 2, Name: "Artist2"},
	}

	relations := []models.Relation{
		{
			ID: 1,
			DatesLocations: map[string][]string{
				"london-uk":    {"01-01-2020", "02-01-2020", "03-01-2020"}, // 3 concerts
				"new_york-usa": {"01-02-2020"},                             // 1 concert
			},
		},
		{
			ID: 2,
			DatesLocations: map[string][]string{
				"london-uk":    {"01-03-2020", "02-03-2020"},                             // 2 more concerts (5 total)
				"paris-france": {"01-04-2020", "02-04-2020", "03-04-2020", "04-04-2020"}, // 4 concerts
				"new_york-usa": {"01-05-2020"},                                           // 1 more concert (2 total)
			},
		},
	}

	// Create simplified mock store
	mockStore := &SimplifiedMockStore{
		artists:   artists,
		relations: relations,
	}

	// Create simplified service
	simplifiedService := service.NewSimplifiedService(mockStore)

	// Test CalculateLocationStats
	locationStats := simplifiedService.CalculateLocationStats()
	if len(locationStats) != 3 {
		t.Errorf("Expected 3 locations, got %d", len(locationStats))
	}

	// Test SortLocationStatsByConcertCount
	sortedStats := simplifiedService.SortLocationStatsByConcertCount(locationStats)

	// Verify sorted order (descending by concert count)
	if len(sortedStats) < 3 {
		t.Fatal("Not enough sorted stats")
	}

	// First should be london-uk with 5 concerts
	if sortedStats[0].Name != "london-uk" || sortedStats[0].ConcertCount != 5 {
		t.Errorf("Expected london-uk with 5 concerts first, got %s with %d concerts",
			sortedStats[0].Name, sortedStats[0].ConcertCount)
	}

	// Second should be paris-france with 4 concerts
	if sortedStats[1].Name != "paris-france" || sortedStats[1].ConcertCount != 4 {
		t.Errorf("Expected paris-france with 4 concerts second, got %s with %d concerts",
			sortedStats[1].Name, sortedStats[1].ConcertCount)
	}

	// Third should be new_york-usa with 2 concerts
	if sortedStats[2].Name != "new_york-usa" || sortedStats[2].ConcertCount != 2 {
		t.Errorf("Expected new_york-usa with 2 concerts third, got %s with %d concerts",
			sortedStats[2].Name, sortedStats[2].ConcertCount)
	}
}

func TestSimplifiedServiceCalculateTotalCountriesIntegration(t *testing.T) {
	mockStore := &SimplifiedMockStore{}
	simplifiedService := service.NewSimplifiedService(mockStore)

	// Create test location stats representing different countries
	locationStats := []service.LocationStat{
		{Name: "london-uk"},
		{Name: "manchester-uk"}, // Same country as london
		{Name: "new_york-usa"},
		{Name: "los_angeles-usa"}, // Same country as new_york
		{Name: "sydney-australia"},
		{Name: "tokyo-japan"},
		{Name: "paris-france"},
	}

	totalCountries := simplifiedService.CalculateTotalCountries(locationStats)

	// Should be 5 unique countries: uk, usa, australia, japan, france
	if totalCountries != 5 {
		t.Errorf("Expected 5 unique countries, got %d", totalCountries)
	}
}

func TestSimplifiedServiceCalculateTotalConcertsIntegration(t *testing.T) {
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
				"tokyo-japan":      {"04-02-1981"},                             // 1 concert
			},
		},
	}

	mockStore := &SimplifiedMockStore{
		relations: relations,
	}

	simplifiedService := service.NewSimplifiedService(mockStore)
	totalConcerts := simplifiedService.CalculateTotalConcerts()

	// Should be 7 total concerts (2 + 1 + 3 + 1)
	if totalConcerts != 7 {
		t.Errorf("Expected 7 total concerts, got %d", totalConcerts)
	}
}
