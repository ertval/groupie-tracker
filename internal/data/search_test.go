package data

import (
	"strings"
	"testing"
)

func TestStore_ArtistLocationSearch(t *testing.T) {
	store := createLocationSearchService()

	tests := []struct {
		name          string
		query         string
		expectedCount int
	}{
		{"Search artists by city - 'london'", "london", 2},
		{"Search artists by country - 'uk'", "uk", 2},
		{"Search artists by combined format - 'london-uk'", "london-uk", 2},
		{"Search artists by USA locations - 'new york'", "new york", 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			params := SearchParams{Query: tt.query}
			result := store.SearchArtists(params)
			if len(result.Artists) != tt.expectedCount {
				t.Errorf("SearchArtists(%q) returned %d artists, expected %d", tt.query, len(result.Artists), tt.expectedCount)
			}
		})
	}
}

func TestStore_SearchArtists(t *testing.T) {
	store := createTestSearchService()

	tests := []struct {
		name     string
		params   SearchParams
		expected []int
	}{
		{
			name:     "Empty query returns all artists",
			params:   SearchParams{Query: ""},
			expected: []int{1, 2, 3},
		},
		{
			name:     "Artist name search - case insensitive",
			params:   SearchParams{Query: "queen"},
			expected: []int{1},
		},
		{
			name:     "Member name search",
			params:   SearchParams{Query: "Freddie Mercury"},
			expected: []int{1},
		},
		{
			name:     "Partial artist name search",
			params:   SearchParams{Query: "Phil"},
			expected: []int{2},
		},
		{
			name:     "Creation year search",
			params:   SearchParams{Query: "1970"},
			expected: []int{1},
		},
		{
			name:     "First album date search",
			params:   SearchParams{Query: "1973"},
			expected: []int{1},
		},
		{
			name:     "No matches returns empty",
			params:   SearchParams{Query: "nonexistent"},
			expected: []int{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := store.SearchArtists(tt.params)

			if len(result.Artists) != len(tt.expected) {
				t.Fatalf("SearchArtists(%q) returned %d artists, expected %d", tt.params.Query, len(result.Artists), len(tt.expected))
			}

			foundIDs := make(map[int]bool)
			for _, artist := range result.Artists {
				foundIDs[artist.ID] = true
			}

			for _, expectedID := range tt.expected {
				if !foundIDs[expectedID] {
					t.Errorf("SearchArtists(%q) missing expected artist ID %d", tt.params.Query, expectedID)
				}
			}

			if result.Query != tt.params.Query {
				t.Errorf("SearchArtists(%q) preserved query as %q", tt.params.Query, result.Query)
			}

			if result.TotalResults != len(result.Artists) {
				t.Errorf("SearchArtists(%q) total results %d mismatch count %d", tt.params.Query, result.TotalResults, len(result.Artists))
			}
		})
	}
}

func TestStore_FilterSearchSuggestions(t *testing.T) {
	store := createTestSearchService()

	t.Run("returns prioritized suggestions", func(t *testing.T) {
		suggestions := store.FilterSearchSuggestions("queen", 5)
		if len(suggestions) == 0 {
			t.Fatal("expected suggestions for query 'queen'")
		}
		if !strings.Contains(strings.ToLower(suggestions[0].Text), "queen") {
			t.Fatalf("expected first suggestion to reference queen, got %q", suggestions[0].Text)
		}
	})

	t.Run("respects max results and handles empty queries", func(t *testing.T) {
		suggestions := store.FilterSearchSuggestions("a", 2)
		if len(suggestions) > 2 {
			t.Fatalf("expected at most 2 suggestions, got %d", len(suggestions))
		}

		empty := store.FilterSearchSuggestions("", 5)
		if len(empty) != 0 {
			t.Fatalf("expected empty result for empty query, got %d entries", len(empty))
		}
	})
}

func TestStore_SearchArtistsCachesSimpleQueries(t *testing.T) {
	store := createTestSearchService()

	params := SearchParams{Query: "queen"}
	store.SearchArtists(params)

	store.searchCacheMu.Lock()
	if len(store.searchCache) != 1 {
		store.searchCacheMu.Unlock()
		t.Fatalf("expected cache size 1, got %d", len(store.searchCache))
	}
	if _, ok := store.searchCache["queen"]; !ok {
		store.searchCacheMu.Unlock()
		t.Fatalf("expected cached entry for 'queen'")
	}
	store.searchCacheMu.Unlock()

	store.SearchArtists(params)

	store.searchCacheMu.Lock()
	defer store.searchCacheMu.Unlock()
	if len(store.searchOrder) != 1 {
		t.Fatalf("expected search order length 1, got %d", len(store.searchOrder))
	}
}

func TestStore_SearchArtistsDoesNotCacheWhenFiltersApplied(t *testing.T) {
	store := createTestSearchService()

	filters := ArtistFilterParams{Countries: []string{"UK"}}
	store.SearchArtists(SearchParams{Query: "queen", Filters: filters})

	store.searchCacheMu.Lock()
	defer store.searchCacheMu.Unlock()
	if len(store.searchCache) != 0 {
		t.Fatalf("expected filtered searches not to be cached, found %d entries", len(store.searchCache))
	}
}

func TestStore_SearchArtistsWithFilters(t *testing.T) {
	store := createTestSearchService()

	tests := []struct {
		name     string
		params   SearchParams
		expected []int
	}{
		{
			name: "Search with creation year filter",
			params: SearchParams{
				Query: "Phil",
				Filters: ArtistFilterParams{
					CreationYearFrom: intPtr(1980),
					CreationYearTo:   intPtr(1985),
				},
			},
			expected: []int{2},
		},
		{
			name: "Search with member count filter",
			params: SearchParams{
				Query:   "",
				Filters: ArtistFilterParams{MemberCounts: []int{1}},
			},
			expected: []int{2},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := store.SearchArtists(tt.params)

			if len(result.Artists) != len(tt.expected) {
				t.Fatalf("SearchArtists returned %d artists, expected %d", len(result.Artists), len(tt.expected))
			}

			foundIDs := make(map[int]bool)
			for _, artist := range result.Artists {
				foundIDs[artist.ID] = true
			}

			for _, expectedID := range tt.expected {
				if !foundIDs[expectedID] {
					t.Errorf("missing expected artist ID %d", expectedID)
				}
			}
		})
	}
}

// Helper utilities for search tests

func createTestSearchService() *Store {
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

	store := NewStoreFromFixtures(artists, nil)
	return store
}

func createLocationSearchService() *Store {
	artists := []Artist{
		{
			ID:           1,
			Name:         "Queen",
			Slug:         "queen",
			Members:      []string{"Freddie Mercury"},
			CreationYear: 1970,
			FirstAlbum:   "14-07-1973",
			Countries:    []string{"UK", "USA"},
			Concerts: []Concert{
				{Date: "01-01-1980", Location: "london-uk"},
				{Date: "01-01-1981", Location: "new-york-usa"},
			},
		},
		{
			ID:           2,
			Name:         "Beatles",
			Slug:         "beatles",
			Members:      []string{"John Lennon"},
			CreationYear: 1960,
			FirstAlbum:   "01-01-1963",
			Countries:    []string{"UK"},
			Concerts:     []Concert{{Date: "01-01-1965", Location: "london-uk"}},
		},
	}

	locations := []Location{{Name: "london-uk", Slug: "london-uk"}, {Name: "new-york-usa", Slug: "new-york-usa"}}
	store := NewStoreFromFixtures(artists, locations)
	return store
}
