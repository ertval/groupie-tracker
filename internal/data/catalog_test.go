package data

import (
	"testing"
	"time"
)

func TestCatalogLocationBuilding(t *testing.T) {
	// Create test artist with concerts
	artist := &Artist{
		ID:           1,
		Name:         "Queen",
		Members:      []string{"Member1"},
		CreationYear: 1970,
		FirstAlbum:   "Queen",
	}

	artist.Concerts = []Concert{
		{
			ArtistID:     1,
			Location:     "london-uk",
			LocationSlug: "london-uk",
			Date:         time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC),
			DateString:   "2022-01-01",
		},
	}

	catalog := NewCatalog()
	catalog.AddArtist(artist)

	if err := catalog.Build(); err != nil {
		t.Fatalf("Failed to build catalog: %v", err)
	}

	// Check that location was created
	locations := catalog.AllLocations()
	if len(locations) != 1 {
		t.Fatalf("Expected 1 location, got %d", len(locations))
	}

	loc := locations[0]
	if loc.Name != "london-uk" {
		t.Errorf("Expected location name 'london-uk', got '%s'", loc.Name)
	}

	if loc.Slug != "london-uk" {
		t.Errorf("Expected location slug 'london-uk', got '%s'", loc.Slug)
	}

	if loc.ArtistCount() != 1 {
		t.Errorf("Expected 1 artist, got %d", loc.ArtistCount())
	}

	// Check LocationBySlug lookup
	loc2, err := catalog.LocationBySlug("london-uk")
	if err != nil {
		t.Fatalf("Failed to find location by slug: %v", err)
	}

	if loc2.Name != loc.Name {
		t.Errorf("LocationBySlug returned different location")
	}

	// Check that artist in location has concerts
	if len(loc.Artists) != 1 {
		t.Fatalf("Expected 1 artist in location, got %d", len(loc.Artists))
	}

	artistAtLoc := loc.Artists[0]
	if artistAtLoc.Artist.Name != "Queen" {
		t.Errorf("Expected artist 'Queen', got '%s'", artistAtLoc.Artist.Name)
	}

	if artistAtLoc.ConcertCount != 1 {
		t.Errorf("Expected 1 concert, got %d", artistAtLoc.ConcertCount)
	}
}

func TestCatalogLocationsSortedByConcertCount(t *testing.T) {
	// Create multiple artists with different concert counts
	artist1 := &Artist{
		ID:           1,
		Name:         "Artist A",
		Members:      []string{"Member 1"},
		CreationYear: 2000,
		FirstAlbum:   "Album A",
	}
	artist1.Concerts = []Concert{
		{ArtistID: 1, Location: "london-uk", LocationSlug: "london-uk", Date: time.Now(), DateString: "2022-01-01"},
		{ArtistID: 1, Location: "london-uk", LocationSlug: "london-uk", Date: time.Now(), DateString: "2022-01-02"},
		{ArtistID: 1, Location: "london-uk", LocationSlug: "london-uk", Date: time.Now(), DateString: "2022-01-03"},
	}

	artist2 := &Artist{
		ID:           2,
		Name:         "Artist B",
		Members:      []string{"Member 2"},
		CreationYear: 2001,
		FirstAlbum:   "Album B",
	}
	artist2.Concerts = []Concert{
		{ArtistID: 2, Location: "paris-france", LocationSlug: "paris-france", Date: time.Now(), DateString: "2022-02-01"},
		{ArtistID: 2, Location: "paris-france", LocationSlug: "paris-france", Date: time.Now(), DateString: "2022-02-02"},
		{ArtistID: 2, Location: "paris-france", LocationSlug: "paris-france", Date: time.Now(), DateString: "2022-02-03"},
		{ArtistID: 2, Location: "paris-france", LocationSlug: "paris-france", Date: time.Now(), DateString: "2022-02-04"},
		{ArtistID: 2, Location: "paris-france", LocationSlug: "paris-france", Date: time.Now(), DateString: "2022-02-05"},
	}

	artist3 := &Artist{
		ID:           3,
		Name:         "Artist C",
		Members:      []string{"Member 3"},
		CreationYear: 2002,
		FirstAlbum:   "Album C",
	}
	artist3.Concerts = []Concert{
		{ArtistID: 3, Location: "tokyo-japan", LocationSlug: "tokyo-japan", Date: time.Now(), DateString: "2022-03-01"},
	}

	catalog := NewCatalog()
	catalog.AddArtist(artist1)
	catalog.AddArtist(artist2)
	catalog.AddArtist(artist3)

	if err := catalog.Build(); err != nil {
		t.Fatalf("Failed to build catalog: %v", err)
	}

	locations := catalog.AllLocations()
	if len(locations) != 3 {
		t.Fatalf("Expected 3 locations, got %d", len(locations))
	}

	// Verify sorting: paris (5 concerts), london (3 concerts), tokyo (1 concert)
	if locations[0].Name != "paris-france" {
		t.Errorf("Expected first location to be 'paris-france', got '%s'", locations[0].Name)
	}
	if locations[0].TotalConcerts() != 5 {
		t.Errorf("Expected paris to have 5 concerts, got %d", locations[0].TotalConcerts())
	}

	if locations[1].Name != "london-uk" {
		t.Errorf("Expected second location to be 'london-uk', got '%s'", locations[1].Name)
	}
	if locations[1].TotalConcerts() != 3 {
		t.Errorf("Expected london to have 3 concerts, got %d", locations[1].TotalConcerts())
	}

	if locations[2].Name != "tokyo-japan" {
		t.Errorf("Expected third location to be 'tokyo-japan', got '%s'", locations[2].Name)
	}
	if locations[2].TotalConcerts() != 1 {
		t.Errorf("Expected tokyo to have 1 concert, got %d", locations[2].TotalConcerts())
	}
}
