package tests

import (
	"context"
	"testing"
	"time"

	"groupie-tracker/internal/api"
	"groupie-tracker/internal/service"
	"groupie-tracker/internal/store"
)

func TestAuditCompliance(t *testing.T) {
	// Test the new refactored architecture
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Initialize services
	apiClient := api.NewClient("https://groupietrackers.herokuapp.com/api", 10*time.Second)
	dataService := service.NewDataService()
	dataStore := store.New()

	// Test API client
	apiArtists, apiRelations, err := apiClient.FetchAllData(ctx)
	if err != nil {
		t.Fatalf("Failed to fetch API data: %v", err)
	}

	if len(apiArtists) == 0 {
		t.Error("No artists fetched from API")
	}

	if len(apiRelations) == 0 {
		t.Error("No relations fetched from API")
	}

	// Test data processing
	artists, locations, stats := dataService.ProcessAPIData(apiArtists, apiRelations)

	if len(artists) == 0 {
		t.Error("No artists processed")
	}

	if len(locations) == 0 {
		t.Error("No locations processed")
	}

	if stats.TotalArtists != len(artists) {
		t.Errorf("Stats mismatch: expected %d artists, got %d", len(artists), stats.TotalArtists)
	}

	// Test data store
	dataStore.LoadData(artists, locations, stats)

	retrievedArtists := dataStore.GetAllArtists()
	if len(retrievedArtists) != len(artists) {
		t.Errorf("Store mismatch: expected %d artists, got %d", len(artists), len(retrievedArtists))
	}

	// Test store indexes
	if len(artists) > 0 {
		firstArtist := artists[0]
		retrieved, exists := dataStore.GetArtistByID(firstArtist.ID)
		if !exists {
			t.Error("Artist not found by ID in store index")
		}
		if retrieved.Name != firstArtist.Name {
			t.Error("Retrieved artist name mismatch")
		}

		retrieved, exists = dataStore.GetArtistBySlug(firstArtist.Slug)
		if !exists {
			t.Error("Artist not found by slug in store index")
		}
		if retrieved.Name != firstArtist.Name {
			t.Error("Retrieved artist name mismatch via slug")
		}
	}
}
