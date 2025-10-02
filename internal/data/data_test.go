package data

import (
	"strconv"
	"strings"
	"testing"
	"time"
)

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
