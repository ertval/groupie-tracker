package data

import (
	"testing"
)

// Test data for search functionality
func createTestSearchData() *Repository {
	// Test artists with various data types for search
	artists := []Artist{
		{
			ID:           1,
			Name:         "Queen",
			Slug:         "queen",
			Members:      []string{"Freddie Mercury", "Brian May", "Roger Taylor", "John Deacon"},
			CreationYear: 1970,
			FirstAlbum:   "14-07-1973",
			Countries:    []string{"USA", "UK"},
		},
		{
			ID:           2,
			Name:         "Phil Collins",
			Slug:         "phil-collins",
			Members:      []string{"Phil Collins"},
			CreationYear: 1981,
			FirstAlbum:   "05-02-1981",
			Countries:    []string{"USA", "UK"},
		},
		{
			ID:           3,
			Name:         "Pink Floyd",
			Slug:         "pink-floyd",
			Members:      []string{"David Gilmour", "Roger Waters", "Nick Mason", "Richard Wright"},
			CreationYear: 1965,
			FirstAlbum:   "05-08-1967",
			Countries:    []string{"USA", "UK", "Germany"},
		},
	}

	// Test locations
	locations := []Location{
		{
			Name: "London UK",
			Slug: "london-uk",
		},
		{
			Name: "New York USA",
			Slug: "new-york-usa",
		},
		{
			Name: "Philadelphia USA",
			Slug: "philadelphia-usa",
		},
	}

	// Create repository with test data
	repo := &Repository{
		artists:         artists,
		artistsByID:     make(map[int]Artist),
		artistsBySlug:   make(map[string]Artist),
		locations:       locations,
		locationsBySlug: make(map[string]Location),
	}

	// Build indexes
	for _, artist := range artists {
		repo.artistsByID[artist.ID] = artist
		repo.artistsBySlug[artist.Slug] = artist
	}

	for _, location := range locations {
		repo.locationsBySlug[location.Slug] = location
	}

	return repo
}

func TestGenerateSearchSuggestions(t *testing.T) {
	repo := createTestSearchData()

	tests := []struct {
		name     string
		query    string
		expected []SearchSuggestion
	}{
		{
			name:     "Empty query returns empty suggestions",
			query:    "",
			expected: []SearchSuggestion{},
		},
		{
			name:  "Artist name match - case insensitive",
			query: "queen",
			expected: []SearchSuggestion{
				{
					Text:        "Queen",
					Type:        SuggestionTypeArtist,
					Description: "Queen - artist",
					URL:         "/artists/queen",
					ArtistID:    1,
				},
			},
		},
		{
			name:  "Member name match - case insensitive",
			query: "freddie",
			expected: []SearchSuggestion{
				{
					Text:        "Freddie Mercury",
					Type:        SuggestionTypeMember,
					Description: "Freddie Mercury - member of Queen",
					URL:         "/artists/queen",
					ArtistID:    1,
				},
			},
		},
		{
			name:  "Phil Collins both artist and member plus location",
			query: "phil",
			expected: []SearchSuggestion{
				{
					Text:        "Phil Collins",
					Type:        SuggestionTypeArtist,
					Description: "Phil Collins - artist",
					URL:         "/artists/phil-collins",
					ArtistID:    2,
				},
				{
					Text:        "Philadelphia USA",
					Type:        SuggestionTypeLocation,
					Description: "Philadelphia USA - location",
					URL:         "/locations/philadelphia-usa",
					ArtistID:    0,
				},
				{
					Text:        "Phil Collins",
					Type:        SuggestionTypeMember,
					Description: "Phil Collins - member of Phil Collins",
					URL:         "/artists/phil-collins",
					ArtistID:    2,
				},
			},
		},
		{
			name:  "Location match",
			query: "london",
			expected: []SearchSuggestion{
				{
					Text:        "London UK",
					Type:        SuggestionTypeLocation,
					Description: "London UK - location",
					URL:         "/locations/london-uk",
					ArtistID:    0,
				},
			},
		},
		{
			name:  "Partial match returns suggestions",
			query: "qu",
			expected: []SearchSuggestion{
				{
					Text:        "Queen",
					Type:        SuggestionTypeArtist,
					Description: "Queen - artist",
					URL:         "/artists/queen",
					ArtistID:    1,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			suggestions := repo.GenerateSearchSuggestions(tt.query)

			// Sort suggestions for consistent comparison
			if len(suggestions) != len(tt.expected) {
				t.Errorf("GenerateSearchSuggestions() got %d suggestions, expected %d", len(suggestions), len(tt.expected))
				t.Logf("Got suggestions: %+v", suggestions)
				t.Logf("Expected suggestions: %+v", tt.expected)
				return
			}

			for i, suggestion := range suggestions {
				if i >= len(tt.expected) {
					break
				}
				expected := tt.expected[i]
				if suggestion.Text != expected.Text ||
					suggestion.Type != expected.Type ||
					suggestion.Description != expected.Description ||
					suggestion.URL != expected.URL ||
					suggestion.ArtistID != expected.ArtistID {
					t.Errorf("GenerateSearchSuggestions() suggestion %d = %+v, expected %+v", i, suggestion, expected)
				}
			}
		})
	}
}

func TestSearchArtists(t *testing.T) {
	repo := createTestSearchData()

	tests := []struct {
		name     string
		params   SearchParams
		expected []int // artist IDs that should be found
	}{
		{
			name: "Empty query returns all artists",
			params: SearchParams{
				Query: "",
			},
			expected: []int{1, 2, 3},
		},
		{
			name: "Artist name search - case insensitive",
			params: SearchParams{
				Query: "queen",
			},
			expected: []int{1},
		},
		{
			name: "Member name search",
			params: SearchParams{
				Query: "Freddie Mercury",
			},
			expected: []int{1},
		},
		{
			name: "Partial artist name search",
			params: SearchParams{
				Query: "Phil",
			},
			expected: []int{2}, // Only Phil Collins
		},
		{
			name: "Location search matches artists who performed there",
			params: SearchParams{
				Query: "London",
			},
			expected: []int{}, // No artists assigned to locations in test data
		},
		{
			name: "Creation year search",
			params: SearchParams{
				Query: "1970",
			},
			expected: []int{1}, // Queen created in 1970
		},
		{
			name: "First album date search",
			params: SearchParams{
				Query: "1973",
			},
			expected: []int{1}, // Queen's first album in 1973
		},
		{
			name: "No matches returns empty",
			params: SearchParams{
				Query: "nonexistent",
			},
			expected: []int{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := repo.SearchArtists(tt.params)

			if len(result.Artists) != len(tt.expected) {
				t.Errorf("SearchArtists() got %d artists, expected %d", len(result.Artists), len(tt.expected))
				return
			}

			// Check if all expected artist IDs are present
			foundIDs := make(map[int]bool)
			for _, artist := range result.Artists {
				foundIDs[artist.ID] = true
			}

			for _, expectedID := range tt.expected {
				if !foundIDs[expectedID] {
					t.Errorf("SearchArtists() missing expected artist ID %d", expectedID)
				}
			}

			// Verify query is preserved
			if result.Query != tt.params.Query {
				t.Errorf("SearchArtists() query = %s, expected %s", result.Query, tt.params.Query)
			}

			// Verify total results count
			if result.TotalResults != len(result.Artists) {
				t.Errorf("SearchArtists() totalResults = %d, expected %d", result.TotalResults, len(result.Artists))
			}
		})
	}
}

func TestSearchArtistsWithFilters(t *testing.T) {
	repo := createTestSearchData()

	tests := []struct {
		name     string
		params   SearchParams
		expected []int // artist IDs
	}{
		{
			name: "Search with creation year filter",
			params: SearchParams{
				Query: "Phil",
				Filters: ArtistFilterParams{
					CreationYearFrom: searchIntPtr(1980),
					CreationYearTo:   searchIntPtr(1985),
				},
			},
			expected: []int{2}, // Only Phil Collins (1981)
		},
		{
			name: "Search with member count filter",
			params: SearchParams{
				Query: "",
				Filters: ArtistFilterParams{
					MemberCounts: []int{1},
				},
			},
			expected: []int{2}, // Only Phil Collins has 1 member
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := repo.SearchArtists(tt.params)

			if len(result.Artists) != len(tt.expected) {
				t.Errorf("SearchArtists() got %d artists, expected %d", len(result.Artists), len(tt.expected))
				return
			}

			foundIDs := make(map[int]bool)
			for _, artist := range result.Artists {
				foundIDs[artist.ID] = true
			}

			for _, expectedID := range tt.expected {
				if !foundIDs[expectedID] {
					t.Errorf("SearchArtists() missing expected artist ID %d", expectedID)
				}
			}
		})
	}
}

func TestNormalizeSearchQuery(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Queen", "queen"},
		{"FREDDIE MERCURY", "freddie mercury"},
		{"  Phil Collins  ", "phil collins"},
		{"New-York", "new-york"},
		{"", ""},
		{"1970", "1970"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := normalizeSearchQuery(tt.input)
			if result != tt.expected {
				t.Errorf("normalizeSearchQuery(%q) = %q, expected %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestMatchesSearchQuery(t *testing.T) {
	artist := Artist{
		Name:         "Queen",
		Members:      []string{"Freddie Mercury", "Brian May"},
		CreationYear: 1970,
		FirstAlbum:   "14-07-1973",
		Countries:    []string{"UK", "USA"},
	}

	tests := []struct {
		query    string
		expected bool
	}{
		{"queen", true},
		{"Queen", true},
		{"QUEEN", true},
		{"freddie", true},
		{"mercury", true},
		{"brian", true},
		{"1970", true},
		{"1973", true},
		{"uk", true},
		{"usa", true},
		{"nonexistent", false},
		{"", true}, // Empty query matches everything
	}

	for _, tt := range tests {
		t.Run(tt.query, func(t *testing.T) {
			result := matchesSearchQuery(artist, normalizeSearchQuery(tt.query))
			if result != tt.expected {
				t.Errorf("matchesSearchQuery(%q) = %t, expected %t", tt.query, result, tt.expected)
			}
		})
	}
}

// Helper function for creating int pointers in tests
func searchIntPtr(i int) *int {
	return &i
}
