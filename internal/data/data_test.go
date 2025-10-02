package data

import (
	"strings"
	"testing"
)

// TestFilterArtists tests all artist filtering functionality with table-driven tests
func TestFilterArtists(t *testing.T) {
	store := createTestStore(t)

	tests := []struct {
		name    string
		params  ArtistFilterParams
		wantMin int
		check   func(t *testing.T, artists []Artist)
	}{
		{
			name: "Filter by creation year range 1995-2000",
			params: ArtistFilterParams{
				CreationYearFrom: intPtr(1995),
				CreationYearTo:   intPtr(2000),
			},
			wantMin: 7,
			check: func(t *testing.T, artists []Artist) {
				expected := []string{"SOJA", "Mamonas Assassinas", "Thirty Seconds to Mars", "Nickelback", "Gorillaz", "Linkin Park", "Coldplay"}
				gotNames := getArtistNames(artists)
				for _, name := range expected {
					if !contains(gotNames, name) {
						t.Errorf("Expected artist %s not found", name)
					}
				}
			},
		},
		{
			name: "Filter by creation year after 2010",
			params: ArtistFilterParams{
				CreationYearFrom: intPtr(2010),
			},
			wantMin: 4,
			check: func(t *testing.T, artists []Artist) {
				expected := []string{"XXXTentacion", "Juice WRLD", "Alec Benjamin", "Post Malone"}
				gotNames := getArtistNames(artists)
				for _, name := range expected {
					if !contains(gotNames, name) {
						t.Errorf("Expected artist %s not found", name)
					}
				}
			},
		},
		{
			name: "Filter by first album year after 2010",
			params: ArtistFilterParams{
				FirstAlbumYearFrom: intPtr(2010),
			},
			wantMin: 4,
		},
		{
			name: "Filter by first album year 1990-1992",
			params: ArtistFilterParams{
				FirstAlbumYearFrom: intPtr(1990),
				FirstAlbumYearTo:   intPtr(1992),
			},
			wantMin: 1,
		},
		{
			name: "Filter solo artists only",
			params: ArtistFilterParams{
				MemberCounts: []int{1},
			},
			wantMin: 2,
			check: func(t *testing.T, artists []Artist) {
				for _, artist := range artists {
					if len(artist.Members) != 1 {
						t.Errorf("Artist %s has %d members, expected 1", artist.Name, len(artist.Members))
					}
				}
			},
		},
		{
			name: "Filter small bands (2-4 members)",
			params: ArtistFilterParams{
				MemberCounts: []int{2, 3, 4},
			},
			wantMin: 5,
			check: func(t *testing.T, artists []Artist) {
				for _, artist := range artists {
					count := len(artist.Members)
					if count < 2 || count > 4 {
						t.Errorf("Artist %s has %d members, expected 2-4", artist.Name, count)
					}
				}
			},
		},
		{
			name: "Filter exactly 7 members",
			params: ArtistFilterParams{
				MemberCounts: []int{7},
			},
			wantMin: 1,
			check: func(t *testing.T, artists []Artist) {
				for _, artist := range artists {
					if len(artist.Members) != 7 {
						t.Errorf("Artist %s has %d members, expected 7", artist.Name, len(artist.Members))
					}
				}
			},
		},
		{
			name: "Filter by country USA",
			params: ArtistFilterParams{
				Countries: []string{"USA"},
			},
			wantMin: 10,
		},
		{
			name: "Filter by country UK",
			params: ArtistFilterParams{
				Countries: []string{"UK"},
			},
			wantMin: 2,
		},
		{
			name: "Filter by countries USA or UK",
			params: ArtistFilterParams{
				Countries: []string{"USA", "UK"},
			},
			wantMin: 12,
		},
		{
			name: "Combined filters - solo artists 1970-2000",
			params: ArtistFilterParams{
				CreationYearFrom: intPtr(1970),
				CreationYearTo:   intPtr(2000),
				MemberCounts:     []int{1},
			},
			wantMin: 0,
		},
		{
			name: "Combined filters - small bands USA concerts after 2000",
			params: ArtistFilterParams{
				CreationYearFrom: intPtr(2000),
				MemberCounts:     []int{2, 3, 4},
				Countries:        []string{"USA"},
			},
			wantMin: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results := store.FilterArtists(tt.params)

			if len(results) < tt.wantMin {
				t.Errorf("FilterArtists() returned %d artists, want at least %d", len(results), tt.wantMin)
				t.Logf("Got artists: %v", getArtistNames(results))
			}

			if tt.check != nil {
				tt.check(t, results)
			}
		})
	}
}

// TestGetArtistFilterOptions tests that filter options are properly computed
func TestGetArtistFilterOptions(t *testing.T) {
	store := createTestStore(t)
	options := store.GetArtistFilterOptions()

	tests := []struct {
		name  string
		check func(t *testing.T)
	}{
		{
			name: "Creation year bounds are set",
			check: func(t *testing.T) {
				if options.CreationYearMin == 0 || options.CreationYearMax == 0 {
					t.Error("Creation year bounds not set properly")
				}
				if options.CreationYearMin >= options.CreationYearMax {
					t.Error("Creation year min should be less than max")
				}
			},
		},
		{
			name: "First album year bounds are set",
			check: func(t *testing.T) {
				if options.FirstAlbumYearMin == 0 || options.FirstAlbumYearMax == 0 {
					t.Error("First album year bounds not set properly")
				}
			},
		},
		{
			name: "Member counts start from 1",
			check: func(t *testing.T) {
				if len(options.MemberCounts) == 0 {
					t.Error("No member counts available")
				}
				if options.MemberCounts[0] != 1 {
					t.Error("Member counts should start from 1")
				}
			},
		},
		{
			name: "Countries list includes USA",
			check: func(t *testing.T) {
				if len(options.Countries) == 0 {
					t.Error("No countries available")
				}
				if !contains(options.Countries, "USA") {
					t.Error("Expected USA to be in countries list")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.check(t)
		})
	}
}

// TestSearchArtists tests artist search functionality with various queries and filters
func TestSearchArtists(t *testing.T) {
	store := createSearchStore()

	tests := []struct {
		name        string
		params      SearchParams
		expectedIDs []int
	}{
		{
			name:        "Empty query returns all artists",
			params:      SearchParams{Query: ""},
			expectedIDs: []int{1, 2, 3},
		},
		{
			name:        "Artist name search - case insensitive",
			params:      SearchParams{Query: "queen"},
			expectedIDs: []int{1},
		},
		{
			name:        "Member name search",
			params:      SearchParams{Query: "Freddie Mercury"},
			expectedIDs: []int{1},
		},
		{
			name:        "Partial artist name search",
			params:      SearchParams{Query: "Phil"},
			expectedIDs: []int{2},
		},
		{
			name:        "Creation year search",
			params:      SearchParams{Query: "1970"},
			expectedIDs: []int{1},
		},
		{
			name:        "First album date search",
			params:      SearchParams{Query: "1973"},
			expectedIDs: []int{1},
		},
		{
			name:        "No matches returns empty",
			params:      SearchParams{Query: "nonexistent"},
			expectedIDs: []int{},
		},
		{
			name: "Search with creation year filter",
			params: SearchParams{
				Query: "Phil",
				Filters: ArtistFilterParams{
					CreationYearFrom: intPtr(1980),
					CreationYearTo:   intPtr(1985),
				},
			},
			expectedIDs: []int{2},
		},
		{
			name: "Search with member count filter",
			params: SearchParams{
				Query:   "",
				Filters: ArtistFilterParams{MemberCounts: []int{1}},
			},
			expectedIDs: []int{2},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := store.SearchArtists(tt.params)

			if len(result.Artists) != len(tt.expectedIDs) {
				t.Fatalf("SearchArtists(%q) returned %d artists, expected %d",
					tt.params.Query, len(result.Artists), len(tt.expectedIDs))
			}

			foundIDs := make(map[int]bool)
			for _, artist := range result.Artists {
				foundIDs[artist.ID] = true
			}

			for _, expectedID := range tt.expectedIDs {
				if !foundIDs[expectedID] {
					t.Errorf("Missing expected artist ID %d", expectedID)
				}
			}

			if result.Query != tt.params.Query {
				t.Errorf("Query mismatch: got %q, want %q", result.Query, tt.params.Query)
			}

			if result.TotalResults != len(result.Artists) {
				t.Errorf("TotalResults %d mismatch actual count %d",
					result.TotalResults, len(result.Artists))
			}
		})
	}
}

// TestSearchArtistsByLocation tests location-based search
func TestSearchArtistsByLocation(t *testing.T) {
	store := createLocationSearchStore()

	tests := []struct {
		name          string
		query         string
		expectedCount int
	}{
		{"Search by city - 'london'", "london", 2},
		{"Search by country - 'uk'", "uk", 2},
		{"Search by combined format - 'london-uk'", "london-uk", 2},
		{"Search by USA locations - 'new york'", "new york", 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			params := SearchParams{Query: tt.query}
			result := store.SearchArtists(params)
			if len(result.Artists) != tt.expectedCount {
				t.Errorf("SearchArtists(%q) returned %d artists, expected %d",
					tt.query, len(result.Artists), tt.expectedCount)
			}
		})
	}
}

// TestSearchSuggestions tests the search suggestions functionality
func TestSearchSuggestions(t *testing.T) {
	store := createSearchStore()

	tests := []struct {
		name       string
		query      string
		maxResults int
		validate   func(t *testing.T, suggestions []SearchSuggestion)
	}{
		{
			name:       "Returns prioritized suggestions for 'queen'",
			query:      "queen",
			maxResults: 5,
			validate: func(t *testing.T, suggestions []SearchSuggestion) {
				if len(suggestions) == 0 {
					t.Fatal("Expected suggestions for query 'queen'")
				}
				if !strings.Contains(strings.ToLower(suggestions[0].Text), "queen") {
					t.Errorf("Expected first suggestion to contain 'queen', got %q", suggestions[0].Text)
				}
			},
		},
		{
			name:       "Respects max results limit",
			query:      "a",
			maxResults: 2,
			validate: func(t *testing.T, suggestions []SearchSuggestion) {
				if len(suggestions) > 2 {
					t.Errorf("Expected at most 2 suggestions, got %d", len(suggestions))
				}
			},
		},
		{
			name:       "Empty query returns no suggestions",
			query:      "",
			maxResults: 5,
			validate: func(t *testing.T, suggestions []SearchSuggestion) {
				if len(suggestions) != 0 {
					t.Errorf("Expected no suggestions for empty query, got %d", len(suggestions))
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			suggestions := store.FilterSearchSuggestions(tt.query, tt.maxResults)
			tt.validate(t, suggestions)
		})
	}
}

// TestSearchCache tests the search result caching mechanism
func TestSearchCache(t *testing.T) {
	store := createSearchStore()

	t.Run("Caches simple queries", func(t *testing.T) {
		params := SearchParams{Query: "queen"}
		store.SearchArtists(params)

		store.searchCacheMu.Lock()
		if len(store.searchCache) != 1 {
			t.Errorf("Expected cache size 1, got %d", len(store.searchCache))
		}
		if _, ok := store.searchCache["queen"]; !ok {
			t.Error("Expected cached entry for 'queen'")
		}
		store.searchCacheMu.Unlock()

		// Search again - should use cache
		store.SearchArtists(params)

		store.searchCacheMu.Lock()
		if len(store.searchOrder) != 1 {
			t.Errorf("Expected search order length 1, got %d", len(store.searchOrder))
		}
		store.searchCacheMu.Unlock()
	})

	t.Run("Does not cache filtered searches", func(t *testing.T) {
		store := createSearchStore() // Fresh store
		filters := ArtistFilterParams{Countries: []string{"UK"}}
		store.SearchArtists(SearchParams{Query: "queen", Filters: filters})

		store.searchCacheMu.Lock()
		defer store.searchCacheMu.Unlock()
		if len(store.searchCache) != 0 {
			t.Errorf("Expected filtered searches not to be cached, found %d entries",
				len(store.searchCache))
		}
	})
}

// Helper functions for creating test stores

func createTestStore(t *testing.T) *Store {
	t.Helper()

	mockArtists := []Artist{
		{ID: 1, Name: "Queen", Members: []string{"Freddie Mercury", "Brian May", "Roger Taylor", "John Deacon", "Mike Grose", "Barry Mitchell", "Doug Fogie"}, CreationYear: 1970, FirstAlbum: "14-07-1973", Concerts: []Concert{{Location: "london-uk"}, {Location: "new-york-usa"}}},
		{ID: 2, Name: "Gorillaz", Members: []string{"Damon Albarn", "Jamie Hewlett"}, CreationYear: 1998, FirstAlbum: "26-03-2001", Concerts: []Concert{{Location: "london-uk"}}},
		{ID: 3, Name: "Travis Scott", Members: []string{"Jacques Berman Webster II"}, CreationYear: 2008, FirstAlbum: "2015", Concerts: []Concert{{Location: "houston-texas-usa"}, {Location: "atlanta-georgia-usa"}, {Location: "chicago-illinois-usa"}, {Location: "los-angeles-california-usa"}, {Location: "miami-florida-usa"}, {Location: "new-york-usa"}, {Location: "philadelphia-pennsylvania-usa"}, {Location: "phoenix-arizona-usa"}}},
		{ID: 4, Name: "Foo Fighters", Members: []string{"Dave Grohl", "Pat Smear", "Chris Shiflett", "Nate Mendel", "Taylor Hawkins", "Rami Jaffee"}, CreationYear: 1994, FirstAlbum: "04-07-1995", Concerts: []Concert{{Location: "seattle-washington-usa"}}},
		{ID: 5, Name: "XXXTentacion", Members: []string{"Jahseh Dwayne Ricardo Onfroy"}, CreationYear: 2013, FirstAlbum: "2017", Concerts: []Concert{{Location: "miami-florida-usa"}}},
		{ID: 6, Name: "Juice WRLD", Members: []string{"Jarad Anthony Higgins"}, CreationYear: 2015, FirstAlbum: "2018", Concerts: []Concert{{Location: "chicago-illinois-usa"}}},
		{ID: 7, Name: "Alec Benjamin", Members: []string{"Alec Shane Benjamin"}, CreationYear: 2013, FirstAlbum: "2018", Concerts: []Concert{{Location: "los-angeles-california-usa"}}},
		{ID: 8, Name: "Post Malone", Members: []string{"Austin Richard Post"}, CreationYear: 2013, FirstAlbum: "2016", Concerts: []Concert{{Location: "new-york-usa"}}},
		{ID: 9, Name: "SOJA", Members: []string{"Jacob Hemphill", "Bob Jefferson", "Patrick O'Shea", "Ryan Berty", "Ken Brownell", "Rafael Rodriguez", "Trevor Young", "Hellman Escorcia"}, CreationYear: 1997, FirstAlbum: "2000", Concerts: []Concert{{Location: "washington-usa"}}},
		{ID: 10, Name: "Mamonas Assassinas", Members: []string{"Dinho", "Júlio Rasec", "Bento Hinoto", "Sérgio Reoli", "Samuel Reoli"}, CreationYear: 1995, FirstAlbum: "1995", Concerts: []Concert{{Location: "sao-paulo-brazil"}}},
		{ID: 11, Name: "Thirty Seconds to Mars", Members: []string{"Jared Leto", "Shannon Leto", "Tomo Miličević"}, CreationYear: 1998, FirstAlbum: "2002", Concerts: []Concert{{Location: "los-angeles-california-usa"}}},
		{ID: 12, Name: "Nickelback", Members: []string{"Chad Kroeger", "Ryan Peake", "Mike Kroeger", "Daniel Adair"}, CreationYear: 1995, FirstAlbum: "1996", Concerts: []Concert{{Location: "vancouver-canada"}}},
		{ID: 13, Name: "Linkin Park", Members: []string{"Chester Bennington", "Mike Shinoda", "Brad Delson", "Dave Farrell", "Joe Hahn", "Rob Bourdon"}, CreationYear: 1996, FirstAlbum: "2000", Concerts: []Concert{{Location: "los-angeles-california-usa"}}},
		{ID: 14, Name: "Coldplay", Members: []string{"Chris Martin", "Guy Berryman", "Jonny Buckland", "Will Champion"}, CreationYear: 1996, FirstAlbum: "2000", Concerts: []Concert{{Location: "london-uk"}}},
		{ID: 15, Name: "Red Hot Chili Peppers", Members: []string{"Anthony Kiedis", "Flea", "Chad Smith", "John Frusciante"}, CreationYear: 1982, FirstAlbum: "1991", Concerts: []Concert{{Location: "los-angeles-california-usa"}}},
	}

	return NewStoreFromFixtures(mockArtists, nil)
}

func createSearchStore() *Store {
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

	return NewStoreFromFixtures(artists, nil)
}

func createLocationSearchStore() *Store {
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

	locations := []Location{
		{Name: "london-uk", Slug: "london-uk"},
		{Name: "new-york-usa", Slug: "new-york-usa"},
	}
	return NewStoreFromFixtures(artists, locations)
}

// Test helper utilities

func getArtistNames(artists []Artist) []string {
	names := make([]string, len(artists))
	for i, artist := range artists {
		names[i] = artist.Name
	}
	return names
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func intPtr(i int) *int {
	return &i
}
