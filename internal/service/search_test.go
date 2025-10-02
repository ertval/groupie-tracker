package service

import (
	"testing"

	"groupie-tracker/internal/data"
)

func TestService_ArtistLocationSearch(t *testing.T) {
	svc := createLocationSearchService()

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
			params := data.SearchParams{Query: tt.query}
			result := svc.SearchArtists(params)
			if len(result.Artists) != tt.expectedCount {
				t.Errorf("SearchArtists(%q) returned %d artists, expected %d", tt.query, len(result.Artists), tt.expectedCount)
			}
		})
	}
}

func TestService_SearchArtists(t *testing.T) {
	svc := createTestSearchService()

	tests := []struct {
		name     string
		params   data.SearchParams
		expected []int
	}{
		{
			name:     "Empty query returns all artists",
			params:   data.SearchParams{Query: ""},
			expected: []int{1, 2, 3},
		},
		{
			name:     "Artist name search - case insensitive",
			params:   data.SearchParams{Query: "queen"},
			expected: []int{1},
		},
		{
			name:     "Member name search",
			params:   data.SearchParams{Query: "Freddie Mercury"},
			expected: []int{1},
		},
		{
			name:     "Partial artist name search",
			params:   data.SearchParams{Query: "Phil"},
			expected: []int{2},
		},
		{
			name:     "Creation year search",
			params:   data.SearchParams{Query: "1970"},
			expected: []int{1},
		},
		{
			name:     "First album date search",
			params:   data.SearchParams{Query: "1973"},
			expected: []int{1},
		},
		{
			name:     "No matches returns empty",
			params:   data.SearchParams{Query: "nonexistent"},
			expected: []int{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := svc.SearchArtists(tt.params)

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

func TestService_SearchArtistsCachesSimpleQueries(t *testing.T) {
	svc := createTestSearchService()

	params := data.SearchParams{Query: "queen"}
	svc.SearchArtists(params)

	svc.cacheMu.Lock()
	if len(svc.searchCache) != 1 {
		svc.cacheMu.Unlock()
		t.Fatalf("expected cache size 1, got %d", len(svc.searchCache))
	}
	if _, ok := svc.searchCache["queen"]; !ok {
		svc.cacheMu.Unlock()
		t.Fatalf("expected cached entry for 'queen'")
	}
	svc.cacheMu.Unlock()

	svc.SearchArtists(params)

	svc.cacheMu.Lock()
	defer svc.cacheMu.Unlock()
	if len(svc.searchOrder) != 1 {
		t.Fatalf("expected search order length 1, got %d", len(svc.searchOrder))
	}
}

func TestService_SearchArtistsDoesNotCacheWhenFiltersApplied(t *testing.T) {
	svc := createTestSearchService()

	filters := data.ArtistFilterParams{Countries: []string{"UK"}}
	svc.SearchArtists(data.SearchParams{Query: "queen", Filters: filters})

	svc.cacheMu.Lock()
	defer svc.cacheMu.Unlock()
	if len(svc.searchCache) != 0 {
		t.Fatalf("expected filtered searches not to be cached, found %d entries", len(svc.searchCache))
	}
}

func TestService_SearchArtistsWithFilters(t *testing.T) {
	svc := createTestSearchService()

	tests := []struct {
		name     string
		params   data.SearchParams
		expected []int
	}{
		{
			name: "Search with creation year filter",
			params: data.SearchParams{
				Query: "Phil",
				Filters: data.ArtistFilterParams{
					CreationYearFrom: intPtr(1980),
					CreationYearTo:   intPtr(1985),
				},
			},
			expected: []int{2},
		},
		{
			name: "Search with member count filter",
			params: data.SearchParams{
				Query:   "",
				Filters: data.ArtistFilterParams{MemberCounts: []int{1}},
			},
			expected: []int{2},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := svc.SearchArtists(tt.params)

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

func createTestSearchService() *Service {
	artists := []data.Artist{
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

	store := data.NewStoreFromFixtures(artists, nil)
	return New(store)
}

func createLocationSearchService() *Service {
	artists := []data.Artist{
		{
			ID:           1,
			Name:         "Queen",
			Slug:         "queen",
			Members:      []string{"Freddie Mercury"},
			CreationYear: 1970,
			FirstAlbum:   "14-07-1973",
			Countries:    []string{"UK", "USA"},
			Concerts: []data.Concert{
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
			Concerts:     []data.Concert{{Date: "01-01-1965", Location: "london-uk"}},
		},
	}

	locations := []data.Location{{Name: "london-uk", Slug: "london-uk"}, {Name: "new-york-usa", Slug: "new-york-usa"}}
	store := data.NewStoreFromFixtures(artists, locations)
	return New(store)
}
