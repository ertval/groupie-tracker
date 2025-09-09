package storage

import (
	"context"
	"testing"
	"time"

	"groupie-tracker/internal/models"
)

// TestNewSimplifiedStore tests the creation of a new simplified store
func TestNewSimplifiedStore(t *testing.T) {
	store := NewSimplifiedStore()

	if store == nil {
		t.Fatal("NewSimplifiedStore() returned nil")
	}

	// Verify initial state
	artists := store.GetAllArtists()
	if len(artists) != 0 {
		t.Errorf("Expected empty store, got %d artists", len(artists))
	}

	// Verify stats
	stats := store.GetStats()
	if stats["artists"] != 0 {
		t.Errorf("Expected 0 artists in stats, got %d", stats["artists"])
	}
}

// TestSimplifiedStoreLoadData tests loading data into the simplified store
func TestSimplifiedStoreLoadData(t *testing.T) {
	store := NewSimplifiedStore()

	// Create test data
	testData := models.APIResponse{
		Artists: []models.Artist{
			{ID: 1, Name: "Test Artist 1", Members: []string{"Member 1", "Member 2"}},
			{ID: 2, Name: "Test Artist 2", Members: []string{"Member 3"}},
		},
		Locations: []models.Location{
			{ID: 1, Locations: []string{"new_york-usa", "london-uk"}},
		},
		Dates: []models.Date{
			{ID: 1, Dates: []string{"01-01-2020", "02-01-2020"}},
		},
		Relations: []models.Relation{
			{
				ID: 1,
				DatesLocations: map[string][]string{
					"new_york-usa": {"01-01-2020"},
					"london-uk":    {"02-01-2020"},
				},
			},
		},
	}

	// Load data
	store.LoadData(testData)

	// Verify data was loaded
	artists := store.GetAllArtists()
	if len(artists) != 2 {
		t.Errorf("Expected 2 artists, got %d", len(artists))
	}

	// Verify artist names
	expectedNames := []string{"Test Artist 1", "Test Artist 2"}
	for i, artist := range artists {
		if artist.Name != expectedNames[i] {
			t.Errorf("Expected artist name %s, got %s", expectedNames[i], artist.Name)
		}
	}

	// Verify stats
	stats := store.GetStats()
	if stats["artists"] != 2 {
		t.Errorf("Expected 2 artists in stats, got %d", stats["artists"])
	}
	if stats["locations"] != 1 {
		t.Errorf("Expected 1 location in stats, got %d", stats["locations"])
	}
}

// TestSimplifiedStoreSearchFunctionality tests the search capabilities
func TestSimplifiedStoreSearchFunctionality(t *testing.T) {
	store := NewSimplifiedStore()

	// Create test data
	testData := models.APIResponse{
		Artists: []models.Artist{
			{ID: 1, Name: "Queen", Members: []string{"Freddie Mercury", "Brian May"}},
			{ID: 2, Name: "Beatles", Members: []string{"John Lennon", "Paul McCartney"}},
			{ID: 3, Name: "Led Zeppelin", Members: []string{"Robert Plant", "Jimmy Page"}},
		},
	}

	store.LoadData(testData)

	// Test search by artist name
	results := store.SearchArtists("Queen")
	if len(results) != 1 {
		t.Errorf("Expected 1 result for 'Queen', got %d", len(results))
	}
	if results[0].Name != "Queen" {
		t.Errorf("Expected 'Queen', got %s", results[0].Name)
	}

	// Test search by member name
	results = store.SearchArtists("Freddie")
	if len(results) != 1 {
		t.Errorf("Expected 1 result for 'Freddie', got %d", len(results))
	}
	if results[0].Name != "Queen" {
		t.Errorf("Expected 'Queen', got %s", results[0].Name)
	}

	// Test case-insensitive search
	results = store.SearchArtists("beatles")
	if len(results) != 1 {
		t.Errorf("Expected 1 result for 'beatles', got %d", len(results))
	}
	if results[0].Name != "Beatles" {
		t.Errorf("Expected 'Beatles', got %s", results[0].Name)
	}

	// Test empty search returns all artists sorted
	results = store.SearchArtists("")
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

// TestSimplifiedStoreFilterByYear tests year filtering functionality
func TestSimplifiedStoreFilterByYear(t *testing.T) {
	store := NewSimplifiedStore()

	// Create test data with different creation years
	testData := models.APIResponse{
		Artists: []models.Artist{
			{ID: 1, Name: "Old Band", CreationYear: 1960},
			{ID: 2, Name: "Medium Band", CreationYear: 1980},
			{ID: 3, Name: "New Band", CreationYear: 2000},
		},
	}

	store.LoadData(testData)

	// Test filter by minimum year
	results := store.FilterArtistsByYear(1970, 0)
	if len(results) != 2 {
		t.Errorf("Expected 2 results for min year 1970, got %d", len(results))
	}

	// Test filter by maximum year
	results = store.FilterArtistsByYear(0, 1990)
	if len(results) != 2 {
		t.Errorf("Expected 2 results for max year 1990, got %d", len(results))
	}

	// Test filter by year range
	results = store.FilterArtistsByYear(1970, 1990)
	if len(results) != 1 {
		t.Errorf("Expected 1 result for year range 1970-1990, got %d", len(results))
	}
	if results[0].Name != "Medium Band" {
		t.Errorf("Expected 'Medium Band', got %s", results[0].Name)
	}
}

// TestSimplifiedStoreGetArtistBySlug tests slug-based artist retrieval
func TestSimplifiedStoreGetArtistBySlug(t *testing.T) {
	store := NewSimplifiedStore()

	// Create test data
	testData := models.APIResponse{
		Artists: []models.Artist{
			{ID: 1, Name: "Queen & The Band"},
			{ID: 2, Name: "AC/DC"},
		},
	}

	store.LoadData(testData)

	// Test getting artist by slug
	artist, found := store.GetArtistBySlug("queen-the-band")
	if !found {
		t.Error("Expected to find artist by slug 'queen-the-band'")
	}
	if artist.Name != "Queen & The Band" {
		t.Errorf("Expected 'Queen & The Band', got %s", artist.Name)
	}

	// Test getting artist by slug with special characters
	artist, found = store.GetArtistBySlug("ac-dc")
	if !found {
		t.Error("Expected to find artist by slug 'ac-dc'")
	}
	if artist.Name != "AC/DC" {
		t.Errorf("Expected 'AC/DC', got %s", artist.Name)
	}

	// Test non-existent slug
	_, found = store.GetArtistBySlug("non-existent")
	if found {
		t.Error("Expected not to find artist with non-existent slug")
	}
}

// TestSimplifiedStoreThreadSafety tests concurrent access to the store
func TestSimplifiedStoreThreadSafety(t *testing.T) {
	store := NewSimplifiedStore()

	// Create test data
	testData := models.APIResponse{
		Artists: []models.Artist{
			{ID: 1, Name: "Test Artist 1"},
			{ID: 2, Name: "Test Artist 2"},
		},
	}

	store.LoadData(testData)

	// Run concurrent reads
	done := make(chan bool, 10)

	// Start multiple goroutines for reading
	for i := 0; i < 10; i++ {
		go func() {
			defer func() { done <- true }()

			// Perform various read operations
			store.GetAllArtists()
			store.SearchArtists("Test")
			store.FilterArtistsByYear(1900, 2100)
			store.GetStats()
			store.GetUniqueLocations()
		}()
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		select {
		case <-done:
			// Good, goroutine completed
		case <-time.After(5 * time.Second):
			t.Fatal("Concurrent read test timed out")
		}
	}
}

// TestSimplifiedStoreWithCache tests cache functionality
func TestSimplifiedStoreWithCache(t *testing.T) {
	// Mock API client
	mockClient := &SimplifiedMockAPIClient{
		data: &models.APIResponse{
			Artists: []models.Artist{
				{ID: 1, Name: "Cached Artist"},
			},
		},
	}

	store := NewSimplifiedStoreWithCache(mockClient)

	// Start cache in background
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Initial load
	err := store.RefreshData(ctx)
	if err != nil {
		t.Fatalf("Failed to refresh data: %v", err)
	}

	// Verify data was loaded
	artists := store.GetAllArtists()
	if len(artists) != 1 {
		t.Errorf("Expected 1 artist, got %d", len(artists))
	}
	if artists[0].Name != "Cached Artist" {
		t.Errorf("Expected 'Cached Artist', got %s", artists[0].Name)
	}
}

// SimplifiedMockAPIClient for testing
type SimplifiedMockAPIClient struct {
	data *models.APIResponse
	err  error
}

func (m *SimplifiedMockAPIClient) FetchAllData(ctx context.Context) (*models.APIResponse, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.data, nil
}

// TestSimplifiedStoreIntegration tests the integration between different components
func TestSimplifiedStoreIntegration(t *testing.T) {
	store := NewSimplifiedStore()

	// Create comprehensive test data
	testData := models.APIResponse{
		Artists: []models.Artist{
			{ID: 1, Name: "Queen", Members: []string{"Freddie Mercury", "Brian May", "Roger Taylor", "John Deacon"}, CreationYear: 1970},
			{ID: 2, Name: "AC/DC", Members: []string{"Angus Young", "Malcolm Young"}, CreationYear: 1973},
		},
		Locations: []models.Location{
			{ID: 1, Locations: []string{"london-uk", "new_york-usa"}},
			{ID: 2, Locations: []string{"sydney-australia", "melbourne-australia"}},
		},
		Dates: []models.Date{
			{ID: 1, Dates: []string{"01-01-1980", "02-01-1980"}},
			{ID: 2, Dates: []string{"01-01-1981", "02-01-1981"}},
		},
		Relations: []models.Relation{
			{
				ID: 1,
				DatesLocations: map[string][]string{
					"london-uk":    {"01-01-1980"},
					"new_york-usa": {"02-01-1980"},
				},
			},
			{
				ID: 2,
				DatesLocations: map[string][]string{
					"sydney-australia":    {"01-01-1981"},
					"melbourne-australia": {"02-01-1981"},
				},
			},
		},
	}

	store.LoadData(testData)

	// Test complete data integrity
	if len(store.GetAllArtists()) != 2 {
		t.Error("Artists not loaded correctly")
	}
	if len(store.GetAllLocations()) != 2 {
		t.Error("Locations not loaded correctly")
	}
	if len(store.GetAllDates()) != 2 {
		t.Error("Dates not loaded correctly")
	}
	if len(store.GetAllRelations()) != 2 {
		t.Error("Relations not loaded correctly")
	}

	// Test unique location extraction
	uniqueLocations := store.GetUniqueLocations()
	expectedLocations := 4 // 2 locations per relation
	if len(uniqueLocations) != expectedLocations {
		t.Errorf("Expected %d unique locations, got %d", expectedLocations, len(uniqueLocations))
	}

	// Test search functionality with comprehensive data
	queenResults := store.SearchArtists("Queen")
	if len(queenResults) != 1 || queenResults[0].Name != "Queen" {
		t.Error("Search by artist name failed")
	}

	freddieResults := store.SearchArtists("Freddie")
	if len(freddieResults) != 1 || freddieResults[0].Name != "Queen" {
		t.Error("Search by member name failed")
	}

	// Test year filtering
	seventiesResults := store.FilterArtistsByYear(1970, 1979)
	if len(seventiesResults) != 2 {
		t.Errorf("Expected 2 artists from 1970s, got %d", len(seventiesResults))
	}
}
