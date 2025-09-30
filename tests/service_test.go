package tests

import (
	"testing"

	"groupie-tracker/internal/api"
	"groupie-tracker/internal/models"
	"groupie-tracker/internal/service"
)

func TestDataService(t *testing.T) {
	dataService := service.NewDataService()

	// Test data processing with sample data
	apiArtists := []api.APIArtist{
		{
			ID:           1,
			Name:         "Test Artist",
			Members:      []string{"Member 1", "Member 2"},
			CreationYear: 2000,
			FirstAlbum:   "2005",
			Image:        "test.jpg",
		},
	}

	apiRelations := []api.APIRelationIndex{
		{
			ID: 1,
			DatesLocations: map[string][]string{
				"new-york-usa": {"01-01-2020"},
				"london-uk":    {"02-02-2020"},
			},
		},
	}

	artists, locations, stats := dataService.ProcessAPIData(apiArtists, apiRelations)

	// Test artist processing
	if len(artists) != 1 {
		t.Errorf("Expected 1 artist, got %d", len(artists))
	}

	artist := artists[0]
	if artist.Name != "Test Artist" {
		t.Errorf("Expected artist name 'Test Artist', got '%s'", artist.Name)
	}

	if len(artist.Concerts) != 2 {
		t.Errorf("Expected 2 concerts, got %d", len(artist.Concerts))
	}

	if len(artist.Countries) != 2 {
		t.Errorf("Expected 2 countries, got %d", len(artist.Countries))
	}

	// Test location processing
	if len(locations) != 2 {
		t.Errorf("Expected 2 locations, got %d", len(locations))
	}

	// Test stats calculation
	if stats.TotalArtists != 1 {
		t.Errorf("Expected 1 total artist, got %d", stats.TotalArtists)
	}

	if stats.TotalLocations != 2 {
		t.Errorf("Expected 2 total locations, got %d", stats.TotalLocations)
	}
}

func TestSearchService(t *testing.T) {
	searchService := service.NewSearchService()

	// Create test data
	artists := []models.Artist{
		{
			ID:           1,
			Name:         "The Beatles",
			Members:      []string{"John Lennon", "Paul McCartney"},
			CreationYear: 1960,
			Countries:    []string{"UK"},
		},
		{
			ID:           2,
			Name:         "Queen",
			Members:      []string{"Freddie Mercury", "Brian May"},
			CreationYear: 1970,
			Countries:    []string{"UK", "USA"},
		},
	}

	// Test basic search
	params := models.SearchParams{Query: "Beatles"}
	result := searchService.Search(artists, params)

	if result.TotalResults != 1 {
		t.Errorf("Expected 1 search result, got %d", result.TotalResults)
	}

	if len(result.Artists) != 1 || result.Artists[0].Name != "The Beatles" {
		t.Error("Search did not return correct artist")
	}

	// Test member search
	params = models.SearchParams{Query: "Freddie"}
	result = searchService.Search(artists, params)

	if result.TotalResults != 1 {
		t.Errorf("Expected 1 search result for member search, got %d", result.TotalResults)
	}

	// Test suggestions generation
	suggestions := searchService.GenerateSuggestions(artists)

	if len(suggestions) == 0 {
		t.Error("No suggestions generated")
	}

	// Verify artist suggestions exist
	foundArtistSuggestion := false
	for _, suggestion := range suggestions {
		if suggestion.Type == models.SuggestionTypeArtist && suggestion.Text == "The Beatles" {
			foundArtistSuggestion = true
			break
		}
	}

	if !foundArtistSuggestion {
		t.Error("Artist suggestion not found")
	}
}

func TestFilterService(t *testing.T) {
	filterService := service.NewFilterService()

	// Create test data
	artists := []models.Artist{
		{
			ID:           1,
			Name:         "Old Band",
			Members:      []string{"Member 1", "Member 2"},
			CreationYear: 1970,
			Countries:    []string{"USA"},
		},
		{
			ID:           2,
			Name:         "New Band",
			Members:      []string{"Member 1", "Member 2", "Member 3"},
			CreationYear: 2000,
			Countries:    []string{"UK"},
		},
	}

	// Test creation year filter
	filters := models.Filters{
		CreationYearMin: 1980,
	}

	filtered := filterService.FilterArtists(artists, filters)
	if len(filtered) != 1 {
		t.Errorf("Expected 1 filtered artist, got %d", len(filtered))
	}

	if filtered[0].Name != "New Band" {
		t.Error("Wrong artist after year filtering")
	}

	// Test member count filter
	filters = models.Filters{
		MemberCounts: []int{3},
	}

	filtered = filterService.FilterArtists(artists, filters)
	if len(filtered) != 1 {
		t.Errorf("Expected 1 filtered artist by member count, got %d", len(filtered))
	}

	// Test filter options generation
	options := filterService.GetFilterOptions(artists)

	if options.CreationYearMin != 1970 {
		t.Errorf("Expected min creation year 1970, got %d", options.CreationYearMin)
	}

	if options.CreationYearMax != 2000 {
		t.Errorf("Expected max creation year 2000, got %d", options.CreationYearMax)
	}

	if len(options.Countries) != 2 {
		t.Errorf("Expected 2 countries in options, got %d", len(options.Countries))
	}
}
