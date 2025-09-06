package storage

import (
	"testing"

	"groupie-tracker/internal/models"
)

func TestStore_AddAndGetArtist(t *testing.T) {
	store := NewStore()
	
	artist := models.Artist{
		ID:           1,
		Name:         "Queen",
		Members:      []string{"Freddie Mercury", "Brian May"},
		CreationYear: 1970,
		FirstAlbum:   "14-12-1973",
	}

	// Test adding artist
	store.AddArtist(artist)

	// Test getting artist by ID
	retrievedArtist, exists := store.GetArtist(1)
	if !exists {
		t.Error("Expected artist to exist, but it doesn't")
	}

	if retrievedArtist.Name != "Queen" {
		t.Errorf("Expected artist name to be Queen, got %s", retrievedArtist.Name)
	}

	// Test getting non-existent artist
	_, exists = store.GetArtist(999)
	if exists {
		t.Error("Expected artist to not exist, but it does")
	}
}

func TestStore_GetAllArtists(t *testing.T) {
	store := NewStore()
	
	artists := []models.Artist{
		{ID: 1, Name: "Queen", Members: []string{"Freddie Mercury"}, CreationYear: 1970},
		{ID: 2, Name: "Gorillaz", Members: []string{"Damon Albarn"}, CreationYear: 1998},
	}

	for _, artist := range artists {
		store.AddArtist(artist)
	}

	allArtists := store.GetAllArtists()
	if len(allArtists) != 2 {
		t.Errorf("Expected 2 artists, got %d", len(allArtists))
	}
}

func TestStore_SearchArtists(t *testing.T) {
	store := NewStore()
	
	artists := []models.Artist{
		{ID: 1, Name: "Queen", Members: []string{"Freddie Mercury", "Brian May"}, CreationYear: 1970},
		{ID: 2, Name: "Gorillaz", Members: []string{"Damon Albarn"}, CreationYear: 1998},
		{ID: 3, Name: "Queen Bee", Members: []string{"Someone"}, CreationYear: 2000},
	}

	for _, artist := range artists {
		store.AddArtist(artist)
	}

	tests := []struct {
		name     string
		query    string
		expected int
	}{
		{"exact match", "Queen", 2},
		{"case insensitive", "queen", 2},
		{"partial match", "Que", 2},
		{"member search", "Freddie", 1},
		{"no match", "Beatles", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results := store.SearchArtists(tt.query)
			if len(results) != tt.expected {
				t.Errorf("Expected %d results for query '%s', got %d", tt.expected, tt.query, len(results))
			}
		})
	}
}

func TestStore_FilterArtistsByYear(t *testing.T) {
	store := NewStore()
	
	artists := []models.Artist{
		{ID: 1, Name: "Queen", CreationYear: 1970},
		{ID: 2, Name: "Gorillaz", CreationYear: 1998},
		{ID: 3, Name: "Modern Band", CreationYear: 2010},
	}

	for _, artist := range artists {
		store.AddArtist(artist)
	}

	// Test filtering by year range
	results := store.FilterArtistsByYear(1990, 2000)
	if len(results) != 1 {
		t.Errorf("Expected 1 artist between 1990-2000, got %d", len(results))
	}

	if results[0].Name != "Gorillaz" {
		t.Errorf("Expected Gorillaz, got %s", results[0].Name)
	}

	// Test with no year restrictions
	results = store.FilterArtistsByYear(0, 0)
	if len(results) != 3 {
		t.Errorf("Expected all 3 artists with no year filter, got %d", len(results))
	}
}

func TestStore_LocationsAndDates(t *testing.T) {
	store := NewStore()

	// Test locations
	location := models.Location{
		ID:        1,
		Locations: []string{"london-uk", "manchester-uk"},
	}
	store.AddLocation(location)

	retrievedLocation, exists := store.GetLocation(1)
	if !exists {
		t.Error("Expected location to exist")
	}

	if len(retrievedLocation.Locations) != 2 {
		t.Errorf("Expected 2 locations, got %d", len(retrievedLocation.Locations))
	}

	// Test dates
	date := models.Date{
		ID:    1,
		Dates: []string{"23-08-2019", "24-08-2019"},
	}
	store.AddDate(date)

	retrievedDate, exists := store.GetDate(1)
	if !exists {
		t.Error("Expected date to exist")
	}

	if len(retrievedDate.Dates) != 2 {
		t.Errorf("Expected 2 dates, got %d", len(retrievedDate.Dates))
	}

	// Test relations
	relation := models.Relation{
		ID: 1,
		DatesLocations: map[string][]string{
			"london-uk": {"23-08-2019", "24-08-2019"},
		},
	}
	store.AddRelation(relation)

	retrievedRelation, exists := store.GetRelation(1)
	if !exists {
		t.Error("Expected relation to exist")
	}

	if len(retrievedRelation.DatesLocations) != 1 {
		t.Errorf("Expected 1 dates-location mapping, got %d", len(retrievedRelation.DatesLocations))
	}
}

func TestStore_GetUniqueLocations(t *testing.T) {
	store := NewStore()

	locations := []models.Location{
		{ID: 1, Locations: []string{"london-uk", "manchester-uk"}},
		{ID: 2, Locations: []string{"london-uk", "new_york-usa"}},
	}

	for _, location := range locations {
		store.AddLocation(location)
	}

	uniqueLocations := store.GetUniqueLocations()
	
	expected := 3 // london-uk, manchester-uk, new_york-usa
	if len(uniqueLocations) != expected {
		t.Errorf("Expected %d unique locations, got %d", expected, len(uniqueLocations))
	}

	// Check if london-uk appears only once
	count := 0
	for _, loc := range uniqueLocations {
		if loc == "london-uk" {
			count++
		}
	}
	if count != 1 {
		t.Errorf("Expected london-uk to appear once, appeared %d times", count)
	}
}

func TestStore_LoadData(t *testing.T) {
	store := NewStore()

	testData := StoreData{
		Artists: []models.Artist{
			{ID: 1, Name: "Queen", CreationYear: 1970},
		},
		Locations: []models.Location{
			{ID: 1, Locations: []string{"london-uk"}},
		},
		Dates: []models.Date{
			{ID: 1, Dates: []string{"23-08-2019"}},
		},
		Relations: []models.Relation{
			{ID: 1, DatesLocations: map[string][]string{"london-uk": {"23-08-2019"}}},
		},
	}

	store.LoadData(testData)

	// Verify data was loaded
	if len(store.GetAllArtists()) != 1 {
		t.Error("Expected 1 artist after loading data")
	}

	_, exists := store.GetLocation(1)
	if !exists {
		t.Error("Expected location to exist after loading data")
	}

	_, exists = store.GetDate(1)
	if !exists {
		t.Error("Expected date to exist after loading data")
	}

	_, exists = store.GetRelation(1)
	if !exists {
		t.Error("Expected relation to exist after loading data")
	}
}

func TestStore_ConcurrentAccess(t *testing.T) {
	store := NewStore()
	
	// Test concurrent writes and reads
	done := make(chan bool, 2)

	// Goroutine 1: Add artists
	go func() {
		for i := 1; i <= 100; i++ {
			artist := models.Artist{
				ID:           i,
				Name:         "Artist",
				CreationYear: 2000 + i,
			}
			store.AddArtist(artist)
		}
		done <- true
	}()

	// Goroutine 2: Read artists
	go func() {
		for i := 0; i < 100; i++ {
			store.GetAllArtists()
		}
		done <- true
	}()

	// Wait for both goroutines to finish
	<-done
	<-done

	// Verify all artists were added
	artists := store.GetAllArtists()
	if len(artists) != 100 {
		t.Errorf("Expected 100 artists, got %d", len(artists))
	}
}
