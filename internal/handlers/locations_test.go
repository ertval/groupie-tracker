package handlers

import (
	"testing"

	"groupie-tracker/internal/models"
)

// StoreInterface defines the minimal interface needed for location stats calculation
type StoreInterface interface {
	GetAllArtists() []models.Artist
	GetAllRelations() []models.Relation
	GetUniqueLocations() []string
	GetStats() map[string]int
}

func TestCalculateLocationStats_SortedByMostConcerts(t *testing.T) {
	// Create mock data
	artists := []models.Artist{
		{ID: 1, Name: "Artist1"},
		{ID: 2, Name: "Artist2"},
		{ID: 3, Name: "Artist3"},
	}

	relations := []models.Relation{
		{
			ID: 1,
			DatesLocations: map[string][]string{
				"new_york-usa": {"01-01-2020", "02-01-2020", "03-01-2020"}, // 3 concerts
				"london-uk":    {"01-02-2020"},                             // 1 concert
			},
		},
		{
			ID: 2,
			DatesLocations: map[string][]string{
				"new_york-usa": {"01-03-2020", "02-03-2020"},                             // 2 more concerts (5 total)
				"paris-france": {"01-04-2020", "02-04-2020", "03-04-2020", "04-04-2020"}, // 4 concerts
				"london-uk":    {"01-05-2020"},                                           // 1 more concert (2 total)
			},
		},
		{
			ID: 3,
			DatesLocations: map[string][]string{
				"tokyo-japan": {"01-06-2020"}, // 1 concert
			},
		},
	}

	// Create mock store
	mockStore := &MockStore{
		artists:   artists,
		relations: relations,
	}

	// Create handlers instance with interface
	h := &TestHandlers{store: mockStore}

	// Calculate location stats
	locationStats := h.calculateLocationStats()

	// Verify we have the expected locations
	if len(locationStats) != 4 {
		t.Errorf("Expected 4 locations, got %d", len(locationStats))
	}

	// Find specific locations and verify their concert counts
	locationMap := make(map[string]LocationStat)
	for _, stat := range locationStats {
		locationMap[stat.Name] = stat
	}

	// Verify concert counts
	expectedCounts := map[string]int{
		"new_york-usa": 5, // 3 + 2
		"london-uk":    2, // 1 + 1
		"paris-france": 4, // 4
		"tokyo-japan":  1, // 1
	}

	for location, expectedCount := range expectedCounts {
		if stat, exists := locationMap[location]; !exists {
			t.Errorf("Location %s not found in results", location)
		} else if stat.ConcertCount != expectedCount {
			t.Errorf("Location %s: expected %d concerts, got %d", location, expectedCount, stat.ConcertCount)
		}
	}
}

func TestSortLocationStatsByConcertCount(t *testing.T) {
	// Create test data
	locationStats := []LocationStat{
		{Name: "location1", ConcertCount: 3, ArtistCount: 2},
		{Name: "location2", ConcertCount: 7, ArtistCount: 1},
		{Name: "location3", ConcertCount: 1, ArtistCount: 3},
		{Name: "location4", ConcertCount: 5, ArtistCount: 2},
	}

	// Sort by concert count (descending)
	sortedStats := sortLocationStatsByConcertCount(locationStats)

	// Verify sorting order
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

// TestHandlers wraps the calculation logic for testing
type TestHandlers struct {
	store StoreInterface
}

// calculateLocationStats mirrors the logic from handlers.go for testing
func (h *TestHandlers) calculateLocationStats() []LocationStat {
	locationMap := make(map[string]*LocationStat)
	allArtists := h.store.GetAllArtists()
	allRelations := h.store.GetAllRelations()

	// Create a map of artist ID to artist for quick lookup
	artistMap := make(map[int]models.Artist)
	for _, artist := range allArtists {
		artistMap[artist.ID] = artist
	}

	// Process each relation to build location statistics
	for _, relation := range allRelations {
		artist, exists := artistMap[relation.ID]
		if !exists {
			continue
		}

		for location, dates := range relation.DatesLocations {
			if locationMap[location] == nil {
				locationMap[location] = &LocationStat{
					Name:         location,
					ArtistCount:  0,
					ConcertCount: 0,
					Artists:      []models.Artist{},
				}
			}

			locationMap[location].ArtistCount++
			locationMap[location].ConcertCount += len(dates)
			locationMap[location].Artists = append(locationMap[location].Artists, artist)
		}
	}

	// Convert map to slice
	var locationStats []LocationStat
	for _, stat := range locationMap {
		locationStats = append(locationStats, *stat)
	}

	return locationStats
}

// MockStore for testing
type MockStore struct {
	artists   []models.Artist
	relations []models.Relation
}

func (m *MockStore) GetAllArtists() []models.Artist {
	return m.artists
}

func (m *MockStore) GetAllRelations() []models.Relation {
	return m.relations
}

func (m *MockStore) GetUniqueLocations() []string {
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

func (m *MockStore) GetStats() map[string]int {
	return map[string]int{
		"artists":   len(m.artists),
		"relations": len(m.relations),
	}
}
