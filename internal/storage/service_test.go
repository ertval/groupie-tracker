package storage

import (
	"testing"

	"groupie-tracker/internal/models"
)

// MockDataReader implements DataReader for testing service layer
type MockDataReader struct {
	artists   []models.Artist
	locations []models.Location
	dates     []models.Date
	relations []models.Relation
	uniqueLoc []string
	uniqueDt  []string
	stats     map[string]int
}

func (m *MockDataReader) GetAllArtists() []models.Artist     { return m.artists }
func (m *MockDataReader) GetAllLocations() []models.Location { return m.locations }
func (m *MockDataReader) GetAllDates() []models.Date         { return m.dates }
func (m *MockDataReader) GetAllRelations() []models.Relation { return m.relations }
func (m *MockDataReader) GetUniqueLocations() []string       { return m.uniqueLoc }
func (m *MockDataReader) GetUniqueDates() []string           { return m.uniqueDt }
func (m *MockDataReader) GetStats() map[string]int           { return m.stats }

func (m *MockDataReader) GetArtist(id int) (models.Artist, bool) {
	for _, artist := range m.artists {
		if artist.ID == id {
			return artist, true
		}
	}
	return models.Artist{}, false
}

func (m *MockDataReader) GetLocation(id int) (models.Location, bool) {
	for _, location := range m.locations {
		if location.ID == id {
			return location, true
		}
	}
	return models.Location{}, false
}

func (m *MockDataReader) GetDate(id int) (models.Date, bool) {
	for _, date := range m.dates {
		if date.ID == id {
			return date, true
		}
	}
	return models.Date{}, false
}

func (m *MockDataReader) GetRelation(id int) (models.Relation, bool) {
	for _, relation := range m.relations {
		if relation.ID == id {
			return relation, true
		}
	}
	return models.Relation{}, false
}

func createMockDataReader() *MockDataReader {
	return &MockDataReader{
		artists: []models.Artist{
			{ID: 1, Name: "Queen", Members: []string{"Freddie Mercury", "Brian May", "Roger Taylor", "John Deacon"}, CreationYear: 1970},
			{ID: 2, Name: "Gorillaz", Members: []string{"Damon Albarn"}, CreationYear: 1998},
			{ID: 3, Name: "Beatles", Members: []string{"John Lennon", "Paul McCartney", "George Harrison", "Ringo Starr"}, CreationYear: 1960},
			{ID: 4, Name: "Arctic Monkeys", Members: []string{"Alex Turner", "Matt Helders"}, CreationYear: 2002},
		},
		locations: []models.Location{
			{ID: 1, Locations: []string{"london-uk", "manchester-uk"}},
			{ID: 2, Locations: []string{"london-uk", "new_york-usa"}},
		},
		relations: []models.Relation{
			{ID: 1, DatesLocations: map[string][]string{"london-uk": {"23-08-2019", "24-08-2019"}, "manchester-uk": {"25-08-2019"}}},
			{ID: 2, DatesLocations: map[string][]string{"london-uk": {"23-09-2019"}, "new_york-usa": {"25-09-2019"}}},
		},
		uniqueLoc: []string{"london-uk", "manchester-uk", "new_york-usa"},
		uniqueDt:  []string{"23-08-2019", "24-08-2019", "25-08-2019"},
		stats:     map[string]int{"artists": 4, "locations": 3, "dates": 3, "relations": 2},
	}
}

func TestNewService(t *testing.T) {
	mockStore := createMockDataReader()
	service := NewService(mockStore)

	if service == nil {
		t.Fatal("NewService() returned nil")
	}

	if service.store != mockStore {
		t.Error("Service store not set correctly")
	}
}

func TestService_SearchArtists(t *testing.T) {
	mockStore := createMockDataReader()
	service := NewService(mockStore)

	tests := []struct {
		name     string
		query    string
		expected int
	}{
		{"empty query returns all", "", 4},
		{"exact match", "Queen", 1},
		{"case insensitive", "queen", 1},
		{"partial match", "Que", 1},
		{"member search", "Freddie", 1},
		{"no match", "Metallica", 0},
		{"multiple matches", "e", 3}, // Queen, Beatles, Arctic Monkeys (all contain 'e')
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results := service.SearchArtists(tt.query)
			if len(results) != tt.expected {
				t.Errorf("Expected %d results for query '%s', got %d", tt.expected, tt.query, len(results))
			}

			// Verify results are sorted alphabetically
			if len(results) > 1 {
				for i := 1; i < len(results); i++ {
					if results[i-1].Name > results[i].Name {
						t.Errorf("Results not sorted alphabetically: %s > %s", results[i-1].Name, results[i].Name)
					}
				}
			}
		})
	}
}

func TestService_FilterArtistsByYear(t *testing.T) {
	mockStore := createMockDataReader()
	service := NewService(mockStore)

	tests := []struct {
		name     string
		minYear  int
		maxYear  int
		expected int
	}{
		{"no filter returns all", 0, 0, 4},
		{"filter by range", 1970, 2000, 2}, // Queen (1970) and Gorillaz (1998)
		{"only min year", 1990, 0, 2},      // Gorillaz (1998) and Arctic Monkeys (2002)
		{"only max year", 0, 1980, 2},      // Queen (1970) and Beatles (1960)
		{"exact year", 1970, 1970, 1},      // Only Queen
		{"no matches", 2010, 2020, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results := service.FilterArtistsByYear(tt.minYear, tt.maxYear)
			if len(results) != tt.expected {
				t.Errorf("Expected %d results for year range %d-%d, got %d", tt.expected, tt.minYear, tt.maxYear, len(results))
			}

			// Verify all results meet the criteria
			for _, artist := range results {
				if tt.minYear > 0 && artist.CreationYear < tt.minYear {
					t.Errorf("Artist %s (%d) doesn't meet min year %d", artist.Name, artist.CreationYear, tt.minYear)
				}
				if tt.maxYear > 0 && artist.CreationYear > tt.maxYear {
					t.Errorf("Artist %s (%d) exceeds max year %d", artist.Name, artist.CreationYear, tt.maxYear)
				}
			}
		})
	}
}

func TestService_FilterArtistsByMemberCount(t *testing.T) {
	mockStore := createMockDataReader()
	service := NewService(mockStore)

	tests := []struct {
		name        string
		memberCount int
		exact       bool
		expected    int
	}{
		{"exactly 1 member", 1, true, 1},    // Gorillaz
		{"exactly 2 members", 2, true, 1},   // Arctic Monkeys
		{"exactly 4 members", 4, true, 2},   // Queen and Beatles
		{"at least 2 members", 2, false, 3}, // Arctic Monkeys, Queen, Beatles
		{"at least 5 members", 5, false, 0}, // None
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results := service.FilterArtistsByMemberCount(tt.memberCount, tt.exact)
			if len(results) != tt.expected {
				t.Errorf("Expected %d results for member count %d (exact=%v), got %d", tt.expected, tt.memberCount, tt.exact, len(results))
			}

			// Verify all results meet the criteria
			for _, artist := range results {
				memberCount := len(artist.Members)
				if tt.exact {
					if memberCount != tt.memberCount {
						t.Errorf("Artist %s has %d members, expected exactly %d", artist.Name, memberCount, tt.memberCount)
					}
				} else {
					if memberCount < tt.memberCount {
						t.Errorf("Artist %s has %d members, expected at least %d", artist.Name, memberCount, tt.memberCount)
					}
				}
			}
		})
	}
}

func TestService_SearchArtistsByLocation(t *testing.T) {
	mockStore := createMockDataReader()
	service := NewService(mockStore)

	tests := []struct {
		name     string
		query    string
		expected int
	}{
		{"empty query returns all", "", 4},
		{"search london", "london", 2}, // Both relations contain london-uk
		{"search uk", "uk", 2},         // Both relations contain UK locations
		{"search usa", "usa", 1},       // Only one relation contains USA
		{"no matches", "paris", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results := service.SearchArtistsByLocation(tt.query)
			if len(results) != tt.expected {
				t.Errorf("Expected %d results for location query '%s', got %d", tt.expected, tt.query, len(results))
			}
		})
	}
}

func TestService_SortingMethods(t *testing.T) {
	mockStore := createMockDataReader()
	service := NewService(mockStore)
	artists := mockStore.GetAllArtists()

	t.Run("SortArtistsByName", func(t *testing.T) {
		sorted := service.SortArtistsByName(artists)
		expected := []string{"Arctic Monkeys", "Beatles", "Gorillaz", "Queen"}

		if len(sorted) != len(expected) {
			t.Errorf("Expected %d artists, got %d", len(expected), len(sorted))
		}

		for i, artist := range sorted {
			if artist.Name != expected[i] {
				t.Errorf("Expected artist %d to be %s, got %s", i, expected[i], artist.Name)
			}
		}
	})

	t.Run("SortArtistsByYear", func(t *testing.T) {
		sorted := service.SortArtistsByYear(artists)
		expected := []int{1960, 1970, 1998, 2002} // Beatles, Queen, Gorillaz, Arctic Monkeys

		for i, artist := range sorted {
			if artist.CreationYear != expected[i] {
				t.Errorf("Expected artist %d to have year %d, got %d", i, expected[i], artist.CreationYear)
			}
		}
	})

	t.Run("SortArtistsByMemberCount", func(t *testing.T) {
		sorted := service.SortArtistsByMemberCount(artists)
		expected := []int{1, 2, 4, 4} // Gorillaz, Arctic Monkeys, Queen, Beatles

		for i, artist := range sorted {
			memberCount := len(artist.Members)
			if memberCount != expected[i] {
				t.Errorf("Expected artist %d to have %d members, got %d", i, expected[i], memberCount)
			}
		}
	})
}

func TestService_GetMostPopularLocations(t *testing.T) {
	mockStore := createMockDataReader()
	service := NewService(mockStore)

	locations := service.GetMostPopularLocations(0) // No limit
	if len(locations) == 0 {
		t.Error("Expected at least one location")
	}

	// Verify sorting by frequency (most frequent first)
	if len(locations) > 1 {
		for i := 1; i < len(locations); i++ {
			if locations[i-1].Count < locations[i].Count {
				t.Errorf("Locations not sorted by frequency: %d < %d", locations[i-1].Count, locations[i].Count)
			}
		}
	}

	// Test with limit
	limitedLocations := service.GetMostPopularLocations(2)
	if len(limitedLocations) > 2 {
		t.Errorf("Expected at most 2 locations, got %d", len(limitedLocations))
	}
}

func TestService_GetDetailedStats(t *testing.T) {
	mockStore := createMockDataReader()
	service := NewService(mockStore)

	stats := service.GetDetailedStats()

	if stats.BasicStats["artists"] != 4 {
		t.Errorf("Expected 4 artists in basic stats, got %d", stats.BasicStats["artists"])
	}

	expectedTotalMembers := 4 + 1 + 4 + 2 // Queen + Gorillaz + Beatles + Arctic Monkeys
	if stats.TotalMembers != expectedTotalMembers {
		t.Errorf("Expected %d total members, got %d", expectedTotalMembers, stats.TotalMembers)
	}

	expectedAvgMembers := float64(expectedTotalMembers) / 4.0
	if stats.AverageMembers != expectedAvgMembers {
		t.Errorf("Expected average members %.2f, got %.2f", expectedAvgMembers, stats.AverageMembers)
	}

	if stats.OldestArtistYear != 1960 {
		t.Errorf("Expected oldest artist year 1960, got %d", stats.OldestArtistYear)
	}

	if stats.NewestArtistYear != 2002 {
		t.Errorf("Expected newest artist year 2002, got %d", stats.NewestArtistYear)
	}

	// Check member count breakdown
	expectedBreakdown := map[int]int{1: 1, 2: 1, 4: 2} // 1 band with 1 member, 1 with 2, 2 with 4
	for count, freq := range expectedBreakdown {
		if stats.MemberCountBreakdown[count] != freq {
			t.Errorf("Expected %d bands with %d members, got %d", freq, count, stats.MemberCountBreakdown[count])
		}
	}
}

func TestService_ImmutabilityOfResults(t *testing.T) {
	mockStore := createMockDataReader()
	service := NewService(mockStore)

	// Get artists and modify the slice
	results1 := service.SearchArtists("")
	originalLen := len(results1)

	// Modify the returned slice
	if len(results1) > 0 {
		results1[0].Name = "Modified Name"
	}

	// Get artists again and verify original data is intact
	results2 := service.SearchArtists("")
	if len(results2) != originalLen {
		t.Error("Original data was modified")
	}

	if len(results2) > 0 && results2[0].Name == "Modified Name" {
		t.Error("Service returned reference to internal data instead of copy")
	}
}
