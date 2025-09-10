package storage

import (
	"context"
	"testing"
	"time"

	"groupie-tracker/internal/models"
)

// TestNewStore tests the creation of a new store
func TestNewStore(t *testing.T) {
	store := NewStore()

	if store == nil {
		t.Fatal("NewStore() returned nil")
	}

	// Verify initial state
	artists := store.GetAllArtists()
	if len(artists) != 0 {
		t.Errorf("Expected empty store, got %d artists", len(artists))
	}
}

// TestStoreLoadData tests loading data into the store
func TestStoreLoadData(t *testing.T) {
	store := NewStore()

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
	expectedNames := map[string]bool{"Test Artist 1": true, "Test Artist 2": true}
	for _, artist := range artists {
		if !expectedNames[artist.Name] {
			t.Errorf("Unexpected artist name: %s", artist.Name)
		}
	}

	// Verify unique locations were computed
	uniqueLocations := store.GetUniqueLocations()
	if len(uniqueLocations) != 2 {
		t.Errorf("Expected 2 unique locations, got %d", len(uniqueLocations))
	}
}

// TestStoreGetArtistBySlug tests slug-based artist retrieval
func TestStoreGetArtistBySlug(t *testing.T) {
	store := NewStore()

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

// TestStoreThreadSafety tests concurrent access to the store
func TestStoreThreadSafety(t *testing.T) {
	store := NewStore()

	// Create test data
	testData := models.APIResponse{
		Artists: []models.Artist{
			{ID: 1, Name: "Test Artist"},
		},
	}

	store.LoadData(testData)

	// Test concurrent reads
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func() {
			defer func() { done <- true }()

			// These operations should not cause race conditions
			store.GetAllArtists()
			store.GetUniqueLocations()
			store.GetUniqueDates()
		}()
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}
}

// TestStoreWithCache tests cache functionality
func TestStoreWithCache(t *testing.T) {
	// Mock API client
	mockClient := &MockAPIClient{
		data: &models.APIResponse{
			Artists: []models.Artist{
				{ID: 1, Name: "Cached Artist"},
			},
		},
	}

	store := NewStoreWithCache(mockClient)

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

// MockAPIClient for testing
type MockAPIClient struct {
	data *models.APIResponse
	err  error
}

func (m *MockAPIClient) FetchAllData(ctx context.Context) (*models.APIResponse, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.data, nil
}

// TestStoreDataIntegrity tests basic data operations
func TestStoreDataIntegrity(t *testing.T) {
	store := NewStore()

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
}
