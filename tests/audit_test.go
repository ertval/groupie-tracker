// Package tests contains integration tests and audit compliance tests.
package tests

import (
	"context"
	"testing"
	"time"

	"groupie-tracker/internal/api"
	"groupie-tracker/internal/models"
	"groupie-tracker/internal/storage"
)

// TestAuditCompliance tests all the specific requirements from audit.md
func TestAuditCompliance(t *testing.T) {
	// Setup: Load real data from API
	store := storage.NewStore()
	client := api.NewClient("https://groupietrackers.herokuapp.com", 30*time.Second)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	data, err := client.FetchAllData(ctx)
	if err != nil {
		t.Fatalf("Failed to load data from API: %v", err)
	}

	storeData := models.APIResponse{
		Artists:   data.Artists,
		Locations: data.Locations,
		Dates:     data.Dates,
		Relations: data.Relations,
	}
	store.LoadData(storeData)

	t.Run("Queen Members Verification", func(t *testing.T) {
		expectedMembers := []string{
			"Freddie Mercury",
			"Brian May",
			"John Daecon",
			"Roger Meddows-Taylor",
			"Mike Grose",
			"Barry Mitchell",
			"Doug Fogie",
		}

		// Find Queen in the artists
		artists := store.GetAllArtists()
		var queen *models.Artist
		for _, artist := range artists {
			if artist.Name == "Queen" {
				queen = &artist
				break
			}
		}

		if queen == nil {
			t.Fatal("Queen not found in the artists data")
		}

		// Verify all expected members are present
		for _, expectedMember := range expectedMembers {
			found := false
			for _, actualMember := range queen.Members {
				if actualMember == expectedMember {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("Expected member '%s' not found in Queen's members list", expectedMember)
			}
		}

		t.Logf("Queen members verified: %v", queen.Members)
	})

	t.Run("Gorillaz First Album Date", func(t *testing.T) {
		expectedDate := "26-03-2001"

		// Find Gorillaz in the artists
		artists := store.GetAllArtists()
		var gorillaz *models.Artist
		for _, artist := range artists {
			if artist.Name == "Gorillaz" {
				gorillaz = &artist
				break
			}
		}

		if gorillaz == nil {
			t.Fatal("Gorillaz not found in the artists data")
		}

		if gorillaz.FirstAlbum != expectedDate {
			t.Errorf("Expected Gorillaz first album date to be '%s', got '%s'", expectedDate, gorillaz.FirstAlbum)
		}

		t.Logf("Gorillaz first album date verified: %s", gorillaz.FirstAlbum)
	})

	t.Run("Travis Scott Locations", func(t *testing.T) {
		expectedLocations := []string{
			"santiago-chile",
			"sao_paulo-brazil", // Note: API uses "brazil" not "brasil"
			"los_angeles-usa",
			"houston-usa",
			"atlanta-usa",
			"new_orleans-usa",
			"philadelphia-usa",
			"london-uk",
			"frauenfeld-switzerland",
			"turku-finland",
		}

		// Find Travis Scott in the artists
		artists := store.GetAllArtists()
		var travisScott *models.Artist
		var travisScottID int
		for _, artist := range artists {
			if artist.Name == "Travis Scott" {
				travisScott = &artist
				travisScottID = artist.ID
				break
			}
		}

		if travisScott == nil {
			t.Fatal("Travis Scott not found in the artists data")
		}

		// Get locations for Travis Scott
		location, exists := store.GetLocation(travisScottID)
		if !exists {
			t.Fatalf("Locations not found for Travis Scott (ID: %d)", travisScottID)
		}

		// Verify all expected locations are present
		for _, expectedLocation := range expectedLocations {
			found := false
			for _, actualLocation := range location.Locations {
				if actualLocation == expectedLocation {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("Expected location '%s' not found in Travis Scott's locations", expectedLocation)
			}
		}

		t.Logf("Travis Scott locations verified: %v", location.Locations)
	})

	t.Run("Foo Fighters Members", func(t *testing.T) {
		expectedMembers := []string{
			"Dave Grohl",
			"Nate Mendel",
			"Taylor Hawkins",
			"Chris Shiflett",
			"Pat Smear",
			"Rami Jaffee",
		}

		// Find Foo Fighters in the artists
		artists := store.GetAllArtists()
		var fooFighters *models.Artist
		for _, artist := range artists {
			if artist.Name == "Foo Fighters" {
				fooFighters = &artist
				break
			}
		}

		if fooFighters == nil {
			t.Fatal("Foo Fighters not found in the artists data")
		}

		// Verify all expected members are present
		for _, expectedMember := range expectedMembers {
			found := false
			for _, actualMember := range fooFighters.Members {
				if actualMember == expectedMember {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("Expected member '%s' not found in Foo Fighters' members list", expectedMember)
			}
		}

		t.Logf("Foo Fighters members verified: %v", fooFighters.Members)
	})
}
