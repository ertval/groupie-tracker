package data

import (
	"strconv"
	"strings"
	"testing"
	"time"
)

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

// makeConcert is a helper to create Concert structs for tests
func makeConcert(artistID int, dateStr, location string) Concert {
	parsedDate, _ := parseDate(dateStr)
	return Concert{
		ArtistID:     artistID,
		Location:     location,
		LocationSlug: createSlug(location),
		Date:         parsedDate,
		DateString:   dateStr,
	}
}

func createSearchStore() *Store {
	artists := []Artist{
		{
			ID:           1,
			Name:         "Queen",
			Members:      []string{"Freddie Mercury", "Brian May", "Roger Taylor", "John Deacon"},
			CreationYear: 1970,
			FirstAlbum:   "14-07-1973",
			Concerts: []Concert{
				makeConcert(1, "14-07-1973", "new-york-usa"),
				makeConcert(1, "15-07-1973", "london-uk"),
			},
		},
		{
			ID:           2,
			Name:         "Phil Collins",
			Members:      []string{"Phil Collins"},
			CreationYear: 1981,
			FirstAlbum:   "05-02-1981",
			Concerts: []Concert{
				makeConcert(2, "05-02-1981", "new-york-usa"),
				makeConcert(2, "06-02-1981", "london-uk"),
			},
		},
		{
			ID:           3,
			Name:         "Pink Floyd",
			Members:      []string{"David Gilmour", "Roger Waters", "Nick Mason", "Richard Wright"},
			CreationYear: 1965,
			FirstAlbum:   "05-08-1967",
			Concerts: []Concert{
				makeConcert(3, "05-08-1967", "new-york-usa"),
				makeConcert(3, "06-08-1967", "london-uk"),
				makeConcert(3, "07-08-1967", "berlin-germany"),
			},
		},
	}

	return NewStoreFromFixtures(artists, nil)
}

func createLocationSearchStore() *Store {
	artists := []Artist{
		{
			ID:           1,
			Name:         "Queen",
			Members:      []string{"Freddie Mercury"},
			CreationYear: 1970,
			FirstAlbum:   "14-07-1973",
			Concerts: []Concert{
				makeConcert(1, "01-01-1980", "london-uk"),
				makeConcert(1, "01-01-1981", "new-york-usa"),
			},
		},
		{
			ID:           2,
			Name:         "Beatles",
			Members:      []string{"John Lennon"},
			CreationYear: 1960,
			FirstAlbum:   "01-01-1963",
			Concerts:     []Concert{makeConcert(2, "01-01-1965", "london-uk")},
		},
	}

	locations := []Location{
		{Name: "london-uk", Slug: "london-uk"},
		{Name: "new-york-usa", Slug: "new-york-usa"},
	}
	return NewStoreFromFixtures(artists, locations)
}

// Test helper utilities

func getArtistNames(artists []*Artist) []string {
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

// TestFilterArtists tests all artist filtering functionality with table-driven tests
func TestFilterArtists(t *testing.T) {
	store := createTestStore(t)

	tests := []struct {
		name    string
		params  ArtistFilterParams
		wantMin int
		check   func(t *testing.T, artists []*Artist)
	}{
		{
			name: "Filter by creation year range 1995-2000",
			params: ArtistFilterParams{
				CreationYearFrom: intPtr(1995),
				CreationYearTo:   intPtr(2000),
			},
			wantMin: 7,
			check: func(t *testing.T, artists []*Artist) {
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
			check: func(t *testing.T, artists []*Artist) {
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
			check: func(t *testing.T, artists []*Artist) {
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
			check: func(t *testing.T, artists []*Artist) {
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
			check: func(t *testing.T, artists []*Artist) {
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

// TestArtistFilterOptions tests that filter options are properly computed
func TestArtistFilterOptions(t *testing.T) {
	store := createTestStore(t)
	options := store.ArtistFilterOptions()

	tests := []struct {
		name  string
		check func(t *testing.T)
	}{
		{
			name: "Creation year bounds are set",
			check: func(t *testing.T) {
				if options.CreationYear.Min == 0 || options.CreationYear.Max == 0 {
					t.Error("Creation year bounds not set properly")
				}
				if options.CreationYear.Min >= options.CreationYear.Max {
					t.Error("Creation year min should be less than max")
				}
			},
		},
		{
			name: "First album year bounds are set",
			check: func(t *testing.T) {
				if options.FirstAlbum.Min == 0 || options.FirstAlbum.Max == 0 {
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

// TestSearchRelevance tests the search relevance ranking
func TestSearchRelevance(t *testing.T) {
	store := createSearchStore()

	t.Run("Exact match ranks first", func(t *testing.T) {
		params := SearchParams{Query: "queen"}
		result := store.SearchArtists(params)

		if len(result.Artists) == 0 {
			t.Fatal("Expected search results for 'queen'")
		}

		// Queen (exact match) should be first
		if result.Artists[0].Name != "Queen" {
			t.Errorf("Expected 'Queen' first, got %s", result.Artists[0].Name)
		}
	})

	t.Run("Prefix match works", func(t *testing.T) {
		params := SearchParams{Query: "phil"}
		result := store.SearchArtists(params)

		if len(result.Artists) == 0 {
			t.Fatal("Expected search results for 'phil'")
		}

		// Phil Collins should be found
		found := false
		for _, artist := range result.Artists {
			if artist.Name == "Phil Collins" {
				found = true
				break
			}
		}
		if !found {
			t.Error("Expected to find Phil Collins in results")
		}
	})

	t.Run("Partial match works", func(t *testing.T) {
		params := SearchParams{Query: "pink"}
		result := store.SearchArtists(params)

		if len(result.Artists) == 0 {
			t.Fatal("Expected search results for 'pink'")
		}

		// Pink Floyd should be found
		found := false
		for _, artist := range result.Artists {
			if artist.Name == "Pink Floyd" {
				found = true
				break
			}
		}
		if !found {
			t.Error("Expected to find Pink Floyd in results")
		}
	})
}

// TestIntRange_Contains tests the IntRange.Contains method
func TestIntRange_Contains(t *testing.T) {
	tests := []struct {
		name     string
		r        IntRange
		value    int
		expected bool
	}{
		{
			name:     "value within range",
			r:        IntRange{Min: 1, Max: 10},
			value:    5,
			expected: true,
		},
		{
			name:     "value at min boundary",
			r:        IntRange{Min: 1, Max: 10},
			value:    1,
			expected: true,
		},
		{
			name:     "value at max boundary",
			r:        IntRange{Min: 1, Max: 10},
			value:    10,
			expected: true,
		},
		{
			name:     "value below range",
			r:        IntRange{Min: 1, Max: 10},
			value:    0,
			expected: false,
		},
		{
			name:     "value above range",
			r:        IntRange{Min: 1, Max: 10},
			value:    11,
			expected: false,
		},
		{
			name:     "negative range",
			r:        IntRange{Min: -10, Max: -5},
			value:    -7,
			expected: true,
		},
		{
			name:     "zero range (single value)",
			r:        IntRange{Min: 0, Max: 0},
			value:    0,
			expected: true,
		},
		{
			name:     "single value range",
			r:        IntRange{Min: 5, Max: 5},
			value:    5,
			expected: true,
		},
		{
			name:     "single value range miss",
			r:        IntRange{Min: 5, Max: 5},
			value:    6,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.r.Contains(tt.value)
			if got != tt.expected {
				t.Errorf("IntRange{%d, %d}.Contains(%d) = %v, want %v",
					tt.r.Min, tt.r.Max, tt.value, got, tt.expected)
			}
		})
	}
}

// TestIntRange_IsZero tests the IntRange.IsZero method
func TestIntRange_IsZero(t *testing.T) {
	tests := []struct {
		name     string
		r        IntRange
		expected bool
	}{
		{
			name:     "zero range",
			r:        IntRange{Min: 0, Max: 0},
			expected: true,
		},
		{
			name:     "non-zero min only",
			r:        IntRange{Min: 1, Max: 0},
			expected: false,
		},
		{
			name:     "non-zero max only",
			r:        IntRange{Min: 0, Max: 1},
			expected: false,
		},
		{
			name:     "both non-zero",
			r:        IntRange{Min: 1, Max: 10},
			expected: false,
		},
		{
			name:     "negative values",
			r:        IntRange{Min: -5, Max: -1},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.r.IsZero()
			if got != tt.expected {
				t.Errorf("IntRange{%d, %d}.IsZero() = %v, want %v",
					tt.r.Min, tt.r.Max, got, tt.expected)
			}
		})
	}
}

// TestStringSet_Contains tests the StringSet.Contains method
func TestStringSet_Contains(t *testing.T) {
	tests := []struct {
		name     string
		s        StringSet
		item     string
		expected bool
	}{
		{
			name:     "item exists",
			s:        NewStringSet("apple", "banana", "cherry"),
			item:     "banana",
			expected: true,
		},
		{
			name:     "item does not exist",
			s:        NewStringSet("apple", "banana", "cherry"),
			item:     "orange",
			expected: false,
		},
		{
			name:     "empty set",
			s:        NewStringSet(),
			item:     "apple",
			expected: false,
		},
		{
			name:     "empty string in set",
			s:        NewStringSet("", "test"),
			item:     "",
			expected: true,
		},
		{
			name:     "case sensitive",
			s:        NewStringSet("Apple", "Banana"),
			item:     "apple",
			expected: false,
		},
		{
			name:     "whitespace matters",
			s:        NewStringSet("test", "test "),
			item:     "test",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.s.Contains(tt.item)
			if got != tt.expected {
				t.Errorf("StringSet.Contains(%q) = %v, want %v",
					tt.item, got, tt.expected)
			}
		})
	}
}

// TestStringSet_IsEmpty tests the StringSet.IsEmpty method
func TestStringSet_IsEmpty(t *testing.T) {
	tests := []struct {
		name     string
		s        StringSet
		expected bool
	}{
		{
			name:     "empty set",
			s:        NewStringSet(),
			expected: true,
		},
		{
			name:     "single item",
			s:        NewStringSet("item"),
			expected: false,
		},
		{
			name:     "multiple items",
			s:        NewStringSet("one", "two", "three"),
			expected: false,
		},
		{
			name:     "nil set",
			s:        nil,
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.s.IsEmpty()
			if got != tt.expected {
				t.Errorf("StringSet.IsEmpty() = %v, want %v", got, tt.expected)
			}
		})
	}
}

// TestIntSet_Contains tests the IntSet.Contains method
func TestIntSet_Contains(t *testing.T) {
	tests := []struct {
		name     string
		s        IntSet
		item     int
		expected bool
	}{
		{
			name:     "item exists",
			s:        NewIntSet(1, 2, 3, 4, 5),
			item:     3,
			expected: true,
		},
		{
			name:     "item does not exist",
			s:        NewIntSet(1, 2, 3, 4, 5),
			item:     10,
			expected: false,
		},
		{
			name:     "empty set",
			s:        NewIntSet(),
			item:     1,
			expected: false,
		},
		{
			name:     "zero in set",
			s:        NewIntSet(0, 1, 2),
			item:     0,
			expected: true,
		},
		{
			name:     "negative numbers",
			s:        NewIntSet(-5, -3, -1, 0, 1),
			item:     -3,
			expected: true,
		},
		{
			name:     "large numbers",
			s:        NewIntSet(1000000, 2000000),
			item:     1000000,
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.s.Contains(tt.item)
			if got != tt.expected {
				t.Errorf("IntSet.Contains(%d) = %v, want %v",
					tt.item, got, tt.expected)
			}
		})
	}
}

// TestIntSet_IsEmpty tests the IntSet.IsEmpty method
func TestIntSet_IsEmpty(t *testing.T) {
	tests := []struct {
		name     string
		s        IntSet
		expected bool
	}{
		{
			name:     "empty set",
			s:        NewIntSet(),
			expected: true,
		},
		{
			name:     "single item",
			s:        NewIntSet(42),
			expected: false,
		},
		{
			name:     "multiple items",
			s:        NewIntSet(1, 2, 3),
			expected: false,
		},
		{
			name:     "nil set",
			s:        nil,
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.s.IsEmpty()
			if got != tt.expected {
				t.Errorf("IntSet.IsEmpty() = %v, want %v", got, tt.expected)
			}
		})
	}
}

// TestNewStringSet tests duplicate handling
func TestNewStringSet(t *testing.T) {
	tests := []struct {
		name     string
		items    []string
		expected int // expected set size
	}{
		{
			name:     "no duplicates",
			items:    []string{"a", "b", "c"},
			expected: 3,
		},
		{
			name:     "with duplicates",
			items:    []string{"a", "b", "a", "c", "b"},
			expected: 3,
		},
		{
			name:     "empty input",
			items:    []string{},
			expected: 0,
		},
		{
			name:     "all duplicates",
			items:    []string{"x", "x", "x"},
			expected: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewStringSet(tt.items...)
			if len(got) != tt.expected {
				t.Errorf("NewStringSet(%v) has %d items, want %d",
					tt.items, len(got), tt.expected)
			}
		})
	}
}

// TestNewIntSet tests duplicate handling
func TestNewIntSet(t *testing.T) {
	tests := []struct {
		name     string
		items    []int
		expected int // expected set size
	}{
		{
			name:     "no duplicates",
			items:    []int{1, 2, 3},
			expected: 3,
		},
		{
			name:     "with duplicates",
			items:    []int{1, 2, 1, 3, 2},
			expected: 3,
		},
		{
			name:     "empty input",
			items:    []int{},
			expected: 0,
		},
		{
			name:     "all duplicates",
			items:    []int{5, 5, 5},
			expected: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewIntSet(tt.items...)
			if len(got) != tt.expected {
				t.Errorf("NewIntSet(%v) has %d items, want %d",
					tt.items, len(got), tt.expected)
			}
		})
	}
}

// TestCreationYearBetween tests the CreationYearBetween filter builder
func TestCreationYearBetween(t *testing.T) {
	tests := []struct {
		name        string
		min         int
		max         int
		artistYear  int
		shouldMatch bool
	}{
		{
			name:        "year within range",
			min:         2000,
			max:         2010,
			artistYear:  2005,
			shouldMatch: true,
		},
		{
			name:        "year at min boundary",
			min:         2000,
			max:         2010,
			artistYear:  2000,
			shouldMatch: true,
		},
		{
			name:        "year at max boundary",
			min:         2000,
			max:         2010,
			artistYear:  2010,
			shouldMatch: true,
		},
		{
			name:        "year below range",
			min:         2000,
			max:         2010,
			artistYear:  1999,
			shouldMatch: false,
		},
		{
			name:        "year above range",
			min:         2000,
			max:         2010,
			artistYear:  2011,
			shouldMatch: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filter := CreationYearBetween(tt.min, tt.max)
			artist := &Artist{CreationYear: tt.artistYear}
			got := filter(artist)
			if got != tt.shouldMatch {
				t.Errorf("CreationYearBetween(%d, %d) for year %d = %v, want %v",
					tt.min, tt.max, tt.artistYear, got, tt.shouldMatch)
			}
		})
	}
}

// TestCreationYearInRange tests zero range handling
func TestCreationYearInRange(t *testing.T) {
	t.Run("zero range matches all", func(t *testing.T) {
		filter := CreationYearInRange(IntRange{Min: 0, Max: 0})
		testYears := []int{1950, 2000, 2020, 0, -100}

		for _, year := range testYears {
			artist := &Artist{CreationYear: year}
			if !filter(artist) {
				t.Errorf("zero range should match year %d", year)
			}
		}
	})
}

// TestHasMemberCount tests the HasMemberCount filter builder
func TestHasMemberCount(t *testing.T) {
	tests := []struct {
		name        string
		counts      []int
		members     []string
		shouldMatch bool
	}{
		{
			name:        "matches single count",
			counts:      []int{4},
			members:     []string{"John", "Paul", "George", "Ringo"},
			shouldMatch: true,
		},
		{
			name:        "matches one of multiple counts",
			counts:      []int{1, 2, 4},
			members:     []string{"John", "Paul", "George", "Ringo"},
			shouldMatch: true,
		},
		{
			name:        "does not match",
			counts:      []int{1, 2, 3},
			members:     []string{"John", "Paul", "George", "Ringo"},
			shouldMatch: false,
		},
		{
			name:        "empty members",
			counts:      []int{0},
			members:     []string{},
			shouldMatch: true,
		},
		{
			name:        "solo artist",
			counts:      []int{1},
			members:     []string{"Solo"},
			shouldMatch: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filter := HasMemberCount(tt.counts...)
			artist := &Artist{Members: tt.members}
			got := filter(artist)
			if got != tt.shouldMatch {
				t.Errorf("HasMemberCount(%v) for %d members = %v, want %v",
					tt.counts, len(tt.members), got, tt.shouldMatch)
			}
		})
	}
}

// TestHasMemberCountInSet tests empty set handling
func TestHasMemberCountInSet(t *testing.T) {
	t.Run("empty set matches all", func(t *testing.T) {
		filter := HasMemberCountInSet(NewIntSet())
		testCounts := []int{0, 1, 4, 10}

		for _, count := range testCounts {
			members := make([]string, count)
			for i := 0; i < count; i++ {
				members[i] = "Member"
			}
			artist := &Artist{Members: members}
			if !filter(artist) {
				t.Errorf("empty set should match member count %d", count)
			}
		}
	})
}

// TestInCountries tests the InCountries filter builder
func TestInCountries(t *testing.T) {
	tests := []struct {
		name        string
		countries   []string
		concerts    []Concert
		shouldMatch bool
	}{
		{
			name:      "matches single country",
			countries: []string{"USA"},
			concerts: []Concert{
				{Location: "new-york-usa", LocationSlug: "new-york-usa"},
				{Location: "los-angeles-usa", LocationSlug: "los-angeles-usa"},
			},
			shouldMatch: true,
		},
		{
			name:      "matches one of multiple countries",
			countries: []string{"UK", "France", "Germany"},
			concerts: []Concert{
				{Location: "paris-france", LocationSlug: "paris-france"},
			},
			shouldMatch: true,
		},
		{
			name:      "does not match",
			countries: []string{"Japan", "Australia"},
			concerts: []Concert{
				{Location: "new-york-usa", LocationSlug: "new-york-usa"},
			},
			shouldMatch: false,
		},
		{
			name:        "no concerts",
			countries:   []string{"USA"},
			concerts:    []Concert{},
			shouldMatch: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filter := InCountries(tt.countries...)
			artist := &Artist{
				ID:       1,
				Concerts: tt.concerts,
			}
			got := filter(artist)
			if got != tt.shouldMatch {
				t.Errorf("InCountries(%v) for %d concerts = %v, want %v",
					tt.countries, len(tt.concerts), got, tt.shouldMatch)
			}
		})
	}
}

// TestInCountrySet tests empty set handling
func TestInCountrySet(t *testing.T) {
	t.Run("empty set matches all", func(t *testing.T) {
		filter := InCountrySet(NewStringSet())

		testCases := [][]Concert{
			{},
			{{Location: "new-york-usa", LocationSlug: "new-york-usa"}},
			{
				{Location: "paris-france", LocationSlug: "paris-france"},
				{Location: "london-uk", LocationSlug: "london-uk"},
			},
		}

		for _, concerts := range testCases {
			artist := &Artist{ID: 1, Concerts: concerts}
			if !filter(artist) {
				t.Errorf("empty set should match any concerts")
			}
		}
	})
}

// TestFirstAlbumYearBetween tests the FirstAlbumYearBetween filter builder
func TestFirstAlbumYearBetween(t *testing.T) {
	tests := []struct {
		name        string
		min         int
		max         int
		albumDate   string
		shouldMatch bool
	}{
		{
			name:        "year within range",
			min:         2000,
			max:         2010,
			albumDate:   "15-03-2005",
			shouldMatch: true,
		},
		{
			name:        "year at min boundary",
			min:         2000,
			max:         2010,
			albumDate:   "01-01-2000",
			shouldMatch: true,
		},
		{
			name:        "year at max boundary",
			min:         2000,
			max:         2010,
			albumDate:   "31-12-2010",
			shouldMatch: true,
		},
		{
			name:        "year below range",
			min:         2000,
			max:         2010,
			albumDate:   "20-06-1999",
			shouldMatch: false,
		},
		{
			name:        "year above range",
			min:         2000,
			max:         2010,
			albumDate:   "10-08-2011",
			shouldMatch: false,
		},
		{
			name:        "no album date (should pass)",
			min:         2000,
			max:         2010,
			albumDate:   "",
			shouldMatch: true,
		},
		{
			name:        "invalid date (should pass)",
			min:         2000,
			max:         2010,
			albumDate:   "invalid",
			shouldMatch: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filter := FirstAlbumYearBetween(tt.min, tt.max)
			artist := &Artist{FirstAlbum: tt.albumDate}
			got := filter(artist)
			if got != tt.shouldMatch {
				t.Errorf("FirstAlbumYearBetween(%d, %d) for date %q = %v, want %v",
					tt.min, tt.max, tt.albumDate, got, tt.shouldMatch)
			}
		})
	}
}

// TestFirstAlbumYearInRange tests zero range handling
func TestFirstAlbumYearInRange(t *testing.T) {
	t.Run("zero range matches all", func(t *testing.T) {
		filter := FirstAlbumYearInRange(IntRange{Min: 0, Max: 0})
		testDates := []string{"01-01-2000", "15-06-1990", "", "invalid"}

		for _, date := range testDates {
			artist := &Artist{FirstAlbum: date}
			if !filter(artist) {
				t.Errorf("zero range should match date %q", date)
			}
		}
	})
}

// TestAndFilters tests the AndFilters combiner
func TestAndFilters(t *testing.T) {
	tests := []struct {
		name        string
		artist      *Artist
		shouldMatch bool
	}{
		{
			name: "matches all filters",
			artist: &Artist{
				ID:           1,
				CreationYear: 2005,
				Members:      []string{"John", "Paul", "George", "Ringo"},
				Concerts: []Concert{
					{Location: "new-york-usa", LocationSlug: "new-york-usa"},
				},
			},
			shouldMatch: true,
		},
		{
			name: "fails creation year",
			artist: &Artist{
				ID:           1,
				CreationYear: 1995,
				Members:      []string{"John", "Paul", "George", "Ringo"},
				Concerts: []Concert{
					{Location: "new-york-usa", LocationSlug: "new-york-usa"},
				},
			},
			shouldMatch: false,
		},
		{
			name: "fails member count",
			artist: &Artist{
				ID:           1,
				CreationYear: 2005,
				Members:      []string{"Solo"},
				Concerts: []Concert{
					{Location: "new-york-usa", LocationSlug: "new-york-usa"},
				},
			},
			shouldMatch: false,
		},
		{
			name: "fails country",
			artist: &Artist{
				ID:           1,
				CreationYear: 2005,
				Members:      []string{"John", "Paul", "George", "Ringo"},
				Concerts: []Concert{
					{Location: "tokyo-japan", LocationSlug: "tokyo-japan"},
				},
			},
			shouldMatch: false,
		},
	}

	// Create combined filter: year 2000-2010, 4 members, in USA
	filter := AndFilters(
		CreationYearBetween(2000, 2010),
		HasMemberCount(4),
		InCountries("USA"),
	)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := filter(tt.artist)
			if got != tt.shouldMatch {
				t.Errorf("AndFilters() = %v, want %v", got, tt.shouldMatch)
			}
		})
	}
}

// TestAndFilters_EmptyList tests empty filter list
func TestAndFilters_EmptyList(t *testing.T) {
	filter := AndFilters()
	artist := &Artist{ID: 1, Name: "Test"}

	if !filter(artist) {
		t.Error("AndFilters with no filters should match all artists")
	}
}

// TestAndFilters_SingleFilter tests single filter
func TestAndFilters_SingleFilter(t *testing.T) {
	filter := AndFilters(CreationYearBetween(2000, 2010))

	tests := []struct {
		year        int
		shouldMatch bool
	}{
		{2005, true},
		{1995, false},
		{2015, false},
	}

	for _, tt := range tests {
		artist := &Artist{CreationYear: tt.year}
		got := filter(artist)
		if got != tt.shouldMatch {
			t.Errorf("year %d: got %v, want %v", tt.year, got, tt.shouldMatch)
		}
	}
}

// --- Empty/Zero-Value Input Tests ---

// TestArtist_EmptyConcerts tests helper methods with no concerts
func TestArtist_EmptyConcerts(t *testing.T) {
	artist := &Artist{
		ID:   1,
		Name: "Test Artist",
	}

	if count := artist.ConcertCount(); count != 0 {
		t.Errorf("ConcertCount() = %d, want 0", count)
	}

	if countries := artist.Countries(); len(countries) != 0 {
		t.Errorf("Countries() = %v, want empty slice", countries)
	}
}

// TestArtist_EmptyMembers tests helper methods with no members
func TestArtist_EmptyMembers(t *testing.T) {
	artist := &Artist{
		ID:   1,
		Name: "Test Artist",
	}

	if count := artist.MemberCount(); count != 0 {
		t.Errorf("MemberCount() = %d, want 0", count)
	}
}

// TestArtist_EmptyFirstAlbum tests FirstAlbumYear with empty string
func TestArtist_EmptyFirstAlbum(t *testing.T) {
	artist := &Artist{
		ID:         1,
		FirstAlbum: "",
	}

	if year := artist.FirstAlbumYear(); year != 0 {
		t.Errorf("FirstAlbumYear() = %d, want 0", year)
	}
}

// TestArtist_InvalidFirstAlbum tests FirstAlbumYear with invalid formats
func TestArtist_InvalidFirstAlbum(t *testing.T) {
	tests := []struct {
		name       string
		firstAlbum string
	}{
		{"malformed date", "invalid"},
		{"garbage", "abc-def-ghij"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			artist := &Artist{
				ID:         1,
				FirstAlbum: tt.firstAlbum,
			}
			year := artist.FirstAlbumYear()
			if year != 0 {
				t.Errorf("FirstAlbumYear() for %q = %d, want 0", tt.firstAlbum, year)
			}
		})
	}
}

// TestArtist_ZeroValues tests an artist with all zero values
func TestArtist_ZeroValues(t *testing.T) {
	artist := &Artist{}

	if artist.MemberCount() != 0 {
		t.Error("zero-value artist should have 0 members")
	}
	if artist.ConcertCount() != 0 {
		t.Error("zero-value artist should have 0 concerts")
	}
	if artist.FirstAlbumYear() != 0 {
		t.Error("zero-value artist should have 0 album year")
	}
	if len(artist.Countries()) != 0 {
		t.Error("zero-value artist should have no countries")
	}
	if artist.Slug() != "" {
		t.Error("zero-value artist should have empty slug")
	}
}

// TestLocation_EmptyConcerts tests location with no concerts
func TestLocation_EmptyConcerts(t *testing.T) {
	loc := &Location{
		Name: "test-location",
	}

	if count := loc.TotalConcerts(); count != 0 {
		t.Errorf("TotalConcerts() = %d, want 0", count)
	}

	if count := loc.ArtistCount(); count != 0 {
		t.Errorf("ArtistCount() = %d, want 0", count)
	}

	minYear, maxYear := loc.YearRange()
	if minYear != 0 || maxYear != 0 {
		t.Errorf("YearRange() = (%d, %d), want (0, 0)", minYear, maxYear)
	}
}

// TestLocation_ZeroValues tests location with all zero values
func TestLocation_ZeroValues(t *testing.T) {
	loc := &Location{}

	if loc.TotalConcerts() != 0 {
		t.Error("zero-value location should have 0 concerts")
	}
	if loc.ArtistCount() != 0 {
		t.Error("zero-value location should have 0 artists")
	}
	if loc.Country() != "" {
		t.Error("zero-value location should have empty country")
	}
	minYear, maxYear := loc.YearRange()
	if minYear != 0 || maxYear != 0 {
		t.Errorf("zero-value location YearRange() = (%d, %d), want (0, 0)", minYear, maxYear)
	}
}

// --- Filter Edge Cases ---

// TestArtistFilters_Empty tests that empty filters match everything
func TestArtistFilters_Empty(t *testing.T) {
	filters := ArtistFilters{}

	if !filters.IsEmpty() {
		t.Error("zero-value ArtistFilters should be empty")
	}

	testArtists := []*Artist{
		{ID: 1, CreationYear: 1960, Members: []string{"Member1"}},
		{ID: 2, CreationYear: 2020, Members: []string{"M1", "M2", "M3", "M4", "M5"}},
		{ID: 3}, // zero-value artist
	}

	for _, artist := range testArtists {
		if !filters.Match(artist) {
			t.Errorf("empty filters should match artist %d", artist.ID)
		}
	}
}

// TestArtistFilters_PartiallySet tests filters with only some criteria set
func TestArtistFilters_PartiallySet(t *testing.T) {
	tests := []struct {
		name    string
		filters ArtistFilters
		artist  *Artist
		match   bool
	}{
		{
			name:    "only creation year set - match",
			filters: ArtistFilters{CreationYear: IntRange{Min: 2000, Max: 2010}},
			artist:  &Artist{CreationYear: 2005, Members: []string{"Any"}},
			match:   true,
		},
		{
			name:    "only creation year set - no match",
			filters: ArtistFilters{CreationYear: IntRange{Min: 2000, Max: 2010}},
			artist:  &Artist{CreationYear: 1990, Members: []string{"Any"}},
			match:   false,
		},
		{
			name:    "only member count set - match",
			filters: ArtistFilters{MemberCounts: NewIntSet(4)},
			artist:  &Artist{CreationYear: 1960, Members: []string{"A", "B", "C", "D"}},
			match:   true,
		},
		{
			name:    "only member count set - no match",
			filters: ArtistFilters{MemberCounts: NewIntSet(4)},
			artist:  &Artist{CreationYear: 1960, Members: []string{"Solo"}},
			match:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.filters.Match(tt.artist)
			if got != tt.match {
				t.Errorf("Match() = %v, want %v", got, tt.match)
			}
		})
	}
}

// TestLocationFilters_Empty tests that empty filters match everything
func TestLocationFilters_Empty(t *testing.T) {
	filters := LocationFilters{}

	if !filters.IsEmpty() {
		t.Error("zero-value LocationFilters should be empty")
	}

	testLocations := []*Location{
		{Name: "test1-usa"},
		{Name: "test2-uk"},
		{}, // zero-value location
	}

	for i, loc := range testLocations {
		if !filters.Match(loc) {
			t.Errorf("empty filters should match location %d", i)
		}
	}
}

// --- Not Found Scenarios ---

// TestCatalog_ArtistByID_NotFound tests looking up non-existent artist ID
func TestCatalog_ArtistByID_NotFound(t *testing.T) {
	catalog := NewCatalog()
	catalog.AddArtist(&Artist{ID: 1, Name: "Test"})
	catalog.Build()

	artist, err := catalog.ArtistByID(999)
	if err == nil {
		t.Error("ArtistByID(999) should return error")
	}
	if artist != nil {
		t.Error("ArtistByID(999) should return nil artist")
	}
}

// TestCatalog_ArtistBySlug_NotFound tests looking up non-existent slug
func TestCatalog_ArtistBySlug_NotFound(t *testing.T) {
	catalog := NewCatalog()
	catalog.AddArtist(&Artist{ID: 1, Name: "Test Artist"})
	catalog.Build()

	artist, err := catalog.ArtistBySlug("nonexistent-slug")
	if err == nil {
		t.Error("ArtistBySlug should return error for non-existent slug")
	}
	if artist != nil {
		t.Error("ArtistBySlug should return nil for non-existent slug")
	}
}

// TestCatalog_LocationBySlug_NotFound tests looking up non-existent location
func TestCatalog_LocationBySlug_NotFound(t *testing.T) {
	catalog := NewCatalog()
	catalog.Locations["test-location"] = Location{Name: "test-location"}
	catalog.Build()

	loc, err := catalog.LocationBySlug("nonexistent-location")
	if err == nil {
		t.Error("LocationBySlug should return error for non-existent slug")
	}
	if loc.Name != "" {
		t.Error("LocationBySlug should return empty location for non-existent slug")
	}
}

// --- Minimal Dataset Tests ---

// TestCatalog_SingleArtist tests catalog with just one artist
func TestCatalog_SingleArtist(t *testing.T) {
	catalog := NewCatalog()
	catalog.AddArtist(&Artist{ID: 1, Name: "Solo Artist", Members: []string{"Solo"}})
	catalog.Build()

	if len(catalog.Artists) != 1 {
		t.Errorf("catalog should have 1 artist, got %d", len(catalog.Artists))
	}

	artist, _ := catalog.ArtistByID(1)
	if artist == nil {
		t.Error("should find the single artist")
	}
}

// TestCatalog_NoArtists tests catalog with no artists
func TestCatalog_NoArtists(t *testing.T) {
	catalog := NewCatalog()
	catalog.Build()

	if len(catalog.Artists) != 0 {
		t.Errorf("empty catalog should have 0 artists, got %d", len(catalog.Artists))
	}

	artist, err := catalog.ArtistByID(1)
	if err == nil {
		t.Error("empty catalog should return error for any ID")
	}
	if artist != nil {
		t.Error("empty catalog should return nil artist")
	}
}

// TestCatalog_NoLocations tests catalog with no locations
func TestCatalog_NoLocations(t *testing.T) {
	catalog := NewCatalog()
	catalog.AddArtist(&Artist{ID: 1, Name: "Artist with no concerts"})
	catalog.Build()

	if len(catalog.Locations) != 0 {
		t.Errorf("catalog should have 0 locations, got %d", len(catalog.Locations))
	}
}

// --- Invalid Input Tests ---

// TestLocation_InvalidLocation tests location name that doesn't parse
func TestLocation_InvalidLocation(t *testing.T) {
	tests := []struct {
		name     string
		location string
	}{
		{"empty string", ""},
		{"just dashes", "---"},
		{"no country code", "city"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			loc := &Location{Name: tt.location}
			country := loc.Country()
			// Should not panic, just return something reasonable
			_ = country
		})
	}
}

// TestConcert_ZeroTime tests concert with zero time value
func TestConcert_ZeroTime(t *testing.T) {
	concert := Concert{
		ArtistID:     1,
		Location:     "test-usa",
		LocationSlug: "test-usa",
		Date:         time.Time{}, // zero time
		DateString:   "",
	}

	// Should not panic when used
	if !concert.Date.IsZero() {
		t.Error("expected zero time")
	}
}

// TestArtist_NegativeCreationYear tests artist with negative year
func TestArtist_NegativeCreationYear(t *testing.T) {
	artist := &Artist{
		ID:           1,
		CreationYear: -1,
	}

	// Should handle gracefully
	filter := CreationYearBetween(2000, 2010)
	if filter(artist) {
		t.Error("negative year should not match positive range")
	}
}

// TestArtist_FutureYear tests artist with future creation year
func TestArtist_FutureYear(t *testing.T) {
	artist := &Artist{
		ID:           1,
		CreationYear: 2100,
	}

	// Should handle gracefully
	filter := CreationYearBetween(2000, 2010)
	if filter(artist) {
		t.Error("future year should not match historical range")
	}
}

// --- Boundary Condition Tests ---

// TestIntRange_InvertedRange tests a range where min > max
func TestIntRange_InvertedRange(t *testing.T) {
	r := IntRange{Min: 10, Max: 1} // inverted

	// Current implementation would not match anything
	if r.Contains(5) {
		t.Error("inverted range should not contain values")
	}
}

// TestStringSet_LargeSet tests set with many items
func TestStringSet_LargeSet(t *testing.T) {
	items := make([]string, 1000)
	for i := 0; i < 1000; i++ {
		items[i] = string(rune('a' + (i % 26)))
	}

	set := NewStringSet(items...)

	// Should handle duplicates efficiently
	if len(set) > 26 {
		t.Errorf("set should deduplicate, got %d items", len(set))
	}
}

// TestIntSet_LargeSet tests set with many items
func TestIntSet_LargeSet(t *testing.T) {
	items := make([]int, 1000)
	for i := 0; i < 1000; i++ {
		items[i] = i % 100
	}

	set := NewIntSet(items...)

	// Should handle duplicates efficiently
	if len(set) > 100 {
		t.Errorf("set should deduplicate, got %d items", len(set))
	}
}

// TestArtistFilters_AllCriteria tests with all filter criteria set
func TestArtistFilters_AllCriteria(t *testing.T) {
	filters := ArtistFilters{
		CreationYear: IntRange{Min: 2000, Max: 2010},
		MemberCounts: NewIntSet(4),
		Countries:    NewStringSet("USA"),
		FirstAlbum:   IntRange{Min: 2005, Max: 2015},
	}

	if filters.IsEmpty() {
		t.Error("filters with all criteria should not be empty")
	}

	// Artist that matches all
	matchingArtist := &Artist{
		ID:           1,
		CreationYear: 2005,
		Members:      []string{"A", "B", "C", "D"},
		FirstAlbum:   "01-01-2010",
		Concerts: []Concert{
			{Location: "new-york-usa"},
		},
	}

	if !filters.Match(matchingArtist) {
		t.Error("artist should match all criteria")
	}

	// Artist that fails one criterion
	nonMatchingArtist := &Artist{
		ID:           2,
		CreationYear: 2005,
		Members:      []string{"A", "B", "C", "D"},
		FirstAlbum:   "01-01-2010",
		Concerts: []Concert{
			{Location: "tokyo-japan"},
		},
	}

	if filters.Match(nonMatchingArtist) {
		t.Error("artist should not match when country doesn't match")
	}
}

// TestArtist_HelperMethods tests Artist domain model helper methods
func TestArtist_HelperMethods(t *testing.T) {
	t.Run("MemberCount", func(t *testing.T) {
		tests := []struct {
			name     string
			artist   Artist
			expected int
		}{
			{"no members", Artist{}, 0},
			{"one member", Artist{Members: []string{"John"}}, 1},
			{"multiple members", Artist{Members: []string{"John", "Paul", "George", "Ringo"}}, 4},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				got := tt.artist.MemberCount()
				if got != tt.expected {
					t.Errorf("MemberCount() = %d, want %d", got, tt.expected)
				}
			})
		}
	})

	t.Run("ConcertCount", func(t *testing.T) {
		tests := []struct {
			name     string
			artist   Artist
			expected int
		}{
			{"no concerts", Artist{}, 0},
			{"one concert", Artist{Concerts: []Concert{{}}}, 1},
			{"multiple concerts", Artist{Concerts: []Concert{{}, {}, {}}}, 3},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				got := tt.artist.ConcertCount()
				if got != tt.expected {
					t.Errorf("ConcertCount() = %d, want %d", got, tt.expected)
				}
			})
		}
	})

	t.Run("FirstAlbumYear", func(t *testing.T) {
		tests := []struct {
			name       string
			firstAlbum string
			expected   int
		}{
			{"valid date DD-MM-YYYY", "05-08-1962", 1962},
			{"valid date DD-MM-YYYY recent", "15-03-2015", 2015},
			{"invalid format", "invalid", 0},
			{"empty string", "", 0},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				artist := Artist{FirstAlbum: tt.firstAlbum}
				got := artist.FirstAlbumYear()
				if got != tt.expected {
					t.Errorf("FirstAlbumYear() = %d, want %d", got, tt.expected)
				}
			})
		}
	})

	t.Run("Countries", func(t *testing.T) {
		tests := []struct {
			name     string
			concerts []Concert
			expected []string
		}{
			{"no concerts", []Concert{}, []string{}},
			{"single country", []Concert{
				{Location: "London-uk"},
				{Location: "Manchester-uk"},
			}, []string{"UK"}},
			{"multiple countries", []Concert{
				{Location: "London-uk"},
				{Location: "New York-usa"},
				{Location: "Paris-france"},
			}, []string{"France", "UK", "USA"}},
			{"duplicate countries", []Concert{
				{Location: "London-uk"},
				{Location: "Manchester-uk"},
				{Location: "New York-usa"},
				{Location: "Los Angeles-usa"},
			}, []string{"UK", "USA"}},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				artist := Artist{Concerts: tt.concerts}
				got := artist.Countries()
				if len(got) != len(tt.expected) {
					t.Errorf("Countries() length = %d, want %d", len(got), len(tt.expected))
				}
				for i, country := range tt.expected {
					if i >= len(got) || got[i] != country {
						t.Errorf("Countries()[%d] = %v, want %v", i, got, tt.expected)
						break
					}
				}
			})
		}
	})

	t.Run("Slug", func(t *testing.T) {
		tests := []struct {
			name     string
			expected string
		}{
			{"Queen", "queen"},
			{"AC/DC", "ac-dc"},
			{"Linkin Park", "linkin-park"},
			{"Twenty One Pilots", "twenty-one-pilots"},
			{"5 Seconds of Summer", "5-seconds-of-summer"},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				artist := Artist{Name: tt.name}
				got := artist.Slug()
				if got != tt.expected {
					t.Errorf("Slug() = %q, want %q", got, tt.expected)
				}
			})
		}
	})

	t.Run("DatesAtLocation", func(t *testing.T) {
		artist := Artist{
			Concerts: []Concert{
				{Location: "London-uk", DateString: "01-01-2020"},
				{Location: "London-uk", DateString: "02-01-2020"},
				{Location: "Paris-france", DateString: "05-01-2020"},
			},
		}

		dates := artist.DatesAtLocation()

		// Check London dates
		londonDates, ok := dates["london-uk"]
		if !ok {
			t.Error("Expected london-uk in dates map")
		}
		if len(londonDates) != 2 {
			t.Errorf("Expected 2 dates for london-uk, got %d", len(londonDates))
		}

		// Check Paris dates
		parisDates, ok := dates["paris-france"]
		if !ok {
			t.Error("Expected paris-france in dates map")
		}
		if len(parisDates) != 1 {
			t.Errorf("Expected 1 date for paris-france, got %d", len(parisDates))
		}
	})
}

// TestLocation_HelperMethods tests Location domain model helper methods
func TestLocation_HelperMethods(t *testing.T) {
	t.Run("Country", func(t *testing.T) {
		tests := []struct {
			name     string
			expected string
		}{
			{"London-uk", "UK"},
			{"New York-usa", "USA"},
			{"Paris-france", "France"},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				loc := Location{Name: tt.name}
				got := loc.Country()
				if got != tt.expected {
					t.Errorf("Country() = %q, want %q", got, tt.expected)
				}
			})
		}
	})

	t.Run("ArtistCount", func(t *testing.T) {
		tests := []struct {
			name     string
			artists  []ArtistAtLocation
			expected int
		}{
			{"no artists", []ArtistAtLocation{}, 0},
			{"one artist", []ArtistAtLocation{{Artist: &Artist{ID: 1}, ConcertCount: 2}}, 1},
			{"multiple artists", []ArtistAtLocation{
				{Artist: &Artist{ID: 1}, ConcertCount: 2},
				{Artist: &Artist{ID: 2}, ConcertCount: 1},
				{Artist: &Artist{ID: 3}, ConcertCount: 3},
			}, 3},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				loc := Location{Artists: tt.artists}
				got := loc.ArtistCount()
				if got != tt.expected {
					t.Errorf("ArtistCount() = %d, want %d", got, tt.expected)
				}
			})
		}
	})

	t.Run("TotalConcerts", func(t *testing.T) {
		loc := Location{
			Artists: []ArtistAtLocation{
				{Artist: &Artist{ID: 1}, ConcertCount: 2},
				{Artist: &Artist{ID: 2}, ConcertCount: 1},
				{Artist: &Artist{ID: 3}, ConcertCount: 3},
			},
		}

		got := loc.TotalConcerts()
		expected := 6 // 2 + 1 + 3

		if got != expected {
			t.Errorf("TotalConcerts() = %d, want %d", got, expected)
		}
	})

	t.Run("YearRange", func(t *testing.T) {
		tests := []struct {
			name         string
			locationName string
			concerts     []Concert
			expectedFrom int
			expectedTo   int
		}{
			{
				name:         "no concerts",
				locationName: "Test Location-country",
				concerts:     []Concert{},
				expectedFrom: 0,
				expectedTo:   0,
			},
			{
				name:         "single year",
				locationName: "London-uk",
				concerts: []Concert{
					{Location: "London-uk", DateString: "15-03-2020", Date: mustParseDate("15-03-2020")},
				},
				expectedFrom: 2020,
				expectedTo:   2020,
			},
			{
				name:         "multiple years",
				locationName: "London-uk",
				concerts: []Concert{
					{Location: "London-uk", DateString: "15-03-2015", Date: mustParseDate("15-03-2015")},
					{Location: "London-uk", DateString: "20-05-2018", Date: mustParseDate("20-05-2018")},
					{Location: "London-uk", DateString: "10-01-2020", Date: mustParseDate("10-01-2020")},
					{Location: "London-uk", DateString: "25-12-2022", Date: mustParseDate("25-12-2022")},
				},
				expectedFrom: 2015,
				expectedTo:   2022,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				// Build artist with concerts for this location
				artist := &Artist{
					ID:       1,
					Name:     "Test Artist",
					Concerts: tt.concerts,
				}
				loc := Location{
					Name:    tt.locationName,
					Slug:    createSlug(tt.locationName),
					Artists: []ArtistAtLocation{{Artist: artist, ConcertCount: len(tt.concerts)}},
				}
				from, to := loc.YearRange()
				if from != tt.expectedFrom || to != tt.expectedTo {
					t.Errorf("YearRange() = (%d, %d), want (%d, %d)", from, to, tt.expectedFrom, tt.expectedTo)
				}
			})
		}
	})
}

// mustParseDate parses a date string or panics (for test use only)
func mustParseDate(dateStr string) time.Time {
	parts := strings.Split(dateStr, "-")
	if len(parts) != 3 {
		panic("invalid date format")
	}

	year, _ := strconv.Atoi(parts[2])
	month, _ := strconv.Atoi(parts[1])
	day, _ := strconv.Atoi(parts[0])
	return time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC)
}

// TestCatalogLocationBuilding tests catalog location building functionality
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

// BenchmarkCatalog_Build benchmarks building the catalog indexes
func BenchmarkCatalog_Build(b *testing.B) {
	// Create a realistic test catalog with multiple artists
	artists := make([]*Artist, 50)
	for i := 0; i < 50; i++ {
		artists[i] = &Artist{
			ID:           i + 1,
			Name:         "Test Artist " + string(rune('A'+i%26)),
			Members:      []string{"Member1", "Member2", "Member3", "Member4"},
			CreationYear: 2000 + i,
			FirstAlbum:   "01-01-2005",
			Concerts: []Concert{
				{
					ArtistID:     i + 1,
					Location:     "new-york-usa",
					LocationSlug: "new-york-usa",
					Date:         time.Now(),
					DateString:   "01-01-2020",
				},
				{
					ArtistID:     i + 1,
					Location:     "paris-france",
					LocationSlug: "paris-france",
					Date:         time.Now(),
					DateString:   "15-06-2021",
				},
			},
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		catalog := NewCatalog()
		for _, artist := range artists {
			catalog.AddArtist(artist)
		}
		catalog.Build()
	}
}

// BenchmarkCatalog_ArtistByID benchmarks artist lookup by ID
func BenchmarkCatalog_ArtistByID(b *testing.B) {
	catalog := setupBenchmarkCatalog()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		catalog.ArtistByID(25)
	}
}

// BenchmarkCatalog_ArtistBySlug benchmarks artist lookup by slug
func BenchmarkCatalog_ArtistBySlug(b *testing.B) {
	catalog := setupBenchmarkCatalog()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		catalog.ArtistBySlug("test-artist-z")
	}
}

// BenchmarkSearchArtists benchmarks search operations
func BenchmarkSearchArtists(b *testing.B) {
	store := setupBenchmarkStore()

	tests := []struct {
		name  string
		query string
	}{
		{"single_word", "test"},
		{"two_words", "test artist"},
		{"partial_match", "art"},
		{"member_search", "member"},
	}

	for _, tt := range tests {
		b.Run(tt.name, func(b *testing.B) {
			params := SearchParams{Query: tt.query}
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				store.SearchArtists(params)
			}
		})
	}
}

// BenchmarkFilterArtists_SingleFilter benchmarks single filter operations
func BenchmarkFilterArtists_SingleFilter(b *testing.B) {
	store := setupBenchmarkStore()

	tests := []struct {
		name    string
		filters ArtistFilters
	}{
		{
			name: "creation_year",
			filters: ArtistFilters{
				CreationYear: IntRange{Min: 2000, Max: 2010},
			},
		},
		{
			name: "member_count",
			filters: ArtistFilters{
				MemberCounts: NewIntSet(4),
			},
		},
		{
			name: "countries",
			filters: ArtistFilters{
				Countries: NewStringSet("USA", "UK"),
			},
		},
		{
			name: "first_album",
			filters: ArtistFilters{
				FirstAlbum: IntRange{Min: 2005, Max: 2015},
			},
		},
	}

	for _, tt := range tests {
		b.Run(tt.name, func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				store.FilterArtistsV2(tt.filters)
			}
		})
	}
}

// BenchmarkFilterArtists_MultipleFilters benchmarks combined filters
func BenchmarkFilterArtists_MultipleFilters(b *testing.B) {
	store := setupBenchmarkStore()

	filters := ArtistFilters{
		CreationYear: IntRange{Min: 2000, Max: 2010},
		MemberCounts: NewIntSet(4),
		Countries:    NewStringSet("USA"),
		FirstAlbum:   IntRange{Min: 2005, Max: 2015},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		store.FilterArtistsV2(filters)
	}
}

// BenchmarkArtist_Countries benchmarks the Countries helper method
func BenchmarkArtist_Countries(b *testing.B) {
	artist := &Artist{
		ID:   1,
		Name: "Test",
		Concerts: []Concert{
			{Location: "new-york-usa"},
			{Location: "paris-france"},
			{Location: "london-uk"},
			{Location: "berlin-germany"},
			{Location: "tokyo-japan"},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		artist.Countries()
	}
}

// BenchmarkArtist_Slug benchmarks the Slug helper method
func BenchmarkArtist_Slug(b *testing.B) {
	tests := []struct {
		name   string
		artist *Artist
	}{
		{
			name:   "simple_name",
			artist: &Artist{Name: "Queen"},
		},
		{
			name:   "special_chars",
			artist: &Artist{Name: "AC/DC"},
		},
		{
			name:   "multiple_words",
			artist: &Artist{Name: "Twenty One Pilots"},
		},
	}

	for _, tt := range tests {
		b.Run(tt.name, func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				tt.artist.Slug()
			}
		})
	}
}

// BenchmarkIntRange_Contains benchmarks range containment check
func BenchmarkIntRange_Contains(b *testing.B) {
	r := IntRange{Min: 2000, Max: 2010}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r.Contains(2005)
	}
}

// BenchmarkStringSet_Contains benchmarks set membership check
func BenchmarkStringSet_Contains(b *testing.B) {
	set := NewStringSet("USA", "UK", "France", "Germany", "Japan")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		set.Contains("USA")
	}
}

// BenchmarkIntSet_Contains benchmarks int set membership check
func BenchmarkIntSet_Contains(b *testing.B) {
	set := NewIntSet(1, 2, 3, 4, 5, 6)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		set.Contains(4)
	}
}

// BenchmarkLocation_TotalConcerts benchmarks concert counting
func BenchmarkLocation_TotalConcerts(b *testing.B) {
	artists := make([]ArtistAtLocation, 10)
	for i := 0; i < 10; i++ {
		artists[i] = ArtistAtLocation{
			Artist:       &Artist{ID: i + 1, Name: "Test"},
			ConcertCount: 10,
		}
	}
	location := &Location{
		Name:    "test-location",
		Artists: artists,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		location.TotalConcerts()
	}
}

// BenchmarkLocation_YearRange benchmarks year range calculation
func BenchmarkLocation_YearRange(b *testing.B) {
	artists := make([]ArtistAtLocation, 10)
	baseTime := time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)
	for i := 0; i < 10; i++ {
		artist := &Artist{
			ID:   i + 1,
			Name: "Test",
			Concerts: []Concert{
				{Date: baseTime.AddDate(i, 0, 0)},
				{Date: baseTime.AddDate(i+10, 0, 0)},
			},
		}
		artists[i] = ArtistAtLocation{
			Artist:       artist,
			ConcertCount: 2,
		}
	}
	location := &Location{
		Name:    "test-usa",
		Artists: artists,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		location.YearRange()
	}
}

// --- Helper functions for benchmarks ---

func setupBenchmarkCatalog() *Catalog {
	catalog := NewCatalog()
	for i := 0; i < 50; i++ {
		catalog.AddArtist(&Artist{
			ID:           i + 1,
			Name:         "Test Artist " + string(rune('A'+i%26)),
			Members:      []string{"Member1", "Member2", "Member3", "Member4"},
			CreationYear: 2000 + i,
			FirstAlbum:   "01-01-2005",
			Concerts: []Concert{
				{Location: "new-york-usa", LocationSlug: "new-york-usa"},
				{Location: "paris-france", LocationSlug: "paris-france"},
			},
		})
	}
	catalog.Build()
	return catalog
}

func setupBenchmarkStore() *Store {
	catalog := setupBenchmarkCatalog()
	store := &Store{
		catalog: catalog,
	}
	return store
}