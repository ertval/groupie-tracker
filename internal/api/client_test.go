package api

import (
	"context"
	"testing"
	"time"
)

func TestClientFetchArtists(t *testing.T) {
	client := NewClient("https://groupietrackers.herokuapp.com", 30*time.Second)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	artists, err := client.FetchArtists(ctx)
	if err != nil {
		t.Fatalf("Failed to fetch artists: %v", err)
	}

	if len(artists) == 0 {
		t.Error("Expected at least one artist")
	}

	// Verify structure of first artist
	if len(artists) > 0 {
		artist := artists[0]
		if artist.ID == 0 {
			t.Error("Expected artist ID to be non-zero")
		}
		if artist.Name == "" {
			t.Error("Expected artist name to be non-empty")
		}
	}
}

func TestClientFetchRelations(t *testing.T) {
	client := NewClient("https://groupietrackers.herokuapp.com", 30*time.Second)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	relations, err := client.FetchRelations(ctx)
	if err != nil {
		t.Fatalf("Failed to fetch relations: %v", err)
	}

	if len(relations.Index) == 0 {
		t.Error("Expected at least one relation")
	}
}

func TestClientTimeout(t *testing.T) {
	// Create client with very short timeout
	client := NewClient("https://groupietrackers.herokuapp.com", 1*time.Millisecond)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	_, err := client.FetchArtists(ctx)
	if err == nil {
		t.Error("Expected timeout error")
	}
}

func TestClientInvalidURL(t *testing.T) {
	client := NewClient("https://invalid-url-that-should-not-exist.com", 5*time.Second)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	_, err := client.FetchArtists(ctx)
	if err == nil {
		t.Error("Expected error for invalid URL")
	}
}

func TestAPIArtistStructure(t *testing.T) {
	// Test that our APIArtist struct correctly maps expected fields
	client := NewClient("https://groupietrackers.herokuapp.com", 30*time.Second)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	artists, err := client.FetchArtists(ctx)
	if err != nil {
		t.Skip("Skipping structure test - API not available")
	}

	if len(artists) == 0 {
		t.Skip("No artists returned from API")
	}

	artist := artists[0]

	// Verify all expected fields are present and valid
	if artist.ID <= 0 {
		t.Errorf("Expected positive ID, got %d", artist.ID)
	}

	if artist.Name == "" {
		t.Error("Expected non-empty name")
	}

	if len(artist.Members) == 0 {
		t.Error("Expected at least one member")
	}

	if artist.CreationDate <= 0 {
		t.Errorf("Expected positive creation date, got %d", artist.CreationDate)
	}

	if artist.FirstAlbum == "" {
		t.Error("Expected non-empty first album")
	}

	if artist.Image == "" {
		t.Error("Expected non-empty image URL")
	}
}

func TestAPIRelationStructure(t *testing.T) {
	client := NewClient("https://groupietrackers.herokuapp.com", 30*time.Second)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	relations, err := client.FetchRelations(ctx)
	if err != nil {
		t.Skip("Skipping structure test - API not available")
	}

	if len(relations.Index) == 0 {
		t.Skip("No relations returned from API")
	}

	relation := relations.Index[0]

	// Verify structure
	if relation.ID <= 0 {
		t.Errorf("Expected positive ID, got %d", relation.ID)
	}

	if len(relation.DatesLocations) == 0 {
		t.Error("Expected at least one date location mapping")
	}

	// Check that dates locations have the expected format
	for location, dates := range relation.DatesLocations {
		if location == "" {
			t.Error("Expected non-empty location")
		}

		if len(dates) == 0 {
			t.Errorf("Expected at least one date for location %s", location)
		}

		for _, date := range dates {
			if date == "" {
				t.Errorf("Expected non-empty date for location %s", location)
			}
		}
	}
}
