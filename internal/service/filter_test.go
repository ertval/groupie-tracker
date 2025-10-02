package service

import (
	"testing"

	"groupie-tracker/internal/data"
)

func TestService_FilterArtists_CreationDate(t *testing.T) {
	tests := []struct {
		name      string
		fromYear  *int
		toYear    *int
		wantNames []string
		wantCount int
	}{
		{
			name:      "Filter 1995-2000",
			fromYear:  intPtr(1995),
			toYear:    intPtr(2000),
			wantNames: []string{"SOJA", "Mamonas Assassinas", "Thirty Seconds to Mars", "Nickelback", "Gorillaz", "Linkin Park", "Coldplay"},
			wantCount: 7,
		},
		{
			name:      "Filter after 2010",
			fromYear:  intPtr(2010),
			toYear:    nil,
			wantNames: []string{"XXXTentacion", "Juice WRLD", "Alec Benjamin", "Post Malone"},
			wantCount: 4,
		},
	}

	svc := createMockService(t)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			params := data.ArtistFilterParams{
				CreationYearFrom: tt.fromYear,
				CreationYearTo:   tt.toYear,
			}

			results := svc.FilterArtists(params)

			if len(results) != tt.wantCount {
				t.Errorf("FilterArtists() returned %d artists, want %d", len(results), tt.wantCount)
				t.Logf("Got artists: %v", getArtistNames(results))
			}

			if len(tt.wantNames) > 0 {
				gotNames := getArtistNames(results)
				for _, wantName := range tt.wantNames {
					if !contains(gotNames, wantName) {
						t.Errorf("Expected artist %s not found in results", wantName)
					}
				}
			}
		})
	}
}

func TestService_FilterArtists_FirstAlbumYear(t *testing.T) {
	tests := []struct {
		name     string
		fromYear *int
		toYear   *int
		wantMin  int
	}{
		{
			name:     "First album after 2010",
			fromYear: intPtr(2010),
			toYear:   nil,
			wantMin:  4,
		},
		{
			name:     "First album 1990-1992",
			fromYear: intPtr(1990),
			toYear:   intPtr(1992),
			wantMin:  1,
		},
	}

	svc := createMockService(t)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			params := data.ArtistFilterParams{
				FirstAlbumYearFrom: tt.fromYear,
				FirstAlbumYearTo:   tt.toYear,
			}

			results := svc.FilterArtists(params)

			if len(results) < tt.wantMin {
				t.Errorf("FilterArtists() returned %d artists, want at least %d", len(results), tt.wantMin)
				t.Logf("Got artists: %v", getArtistNames(results))
			}
		})
	}
}

func TestService_FilterArtists_MemberCounts(t *testing.T) {
	tests := []struct {
		name         string
		memberCounts []int
		wantMin      int
	}{
		{
			name:         "Solo artists only",
			memberCounts: []int{1},
			wantMin:      2,
		},
		{
			name:         "Small bands (2-4 members)",
			memberCounts: []int{2, 3, 4},
			wantMin:      5,
		},
		{
			name:         "Exactly 7 members",
			memberCounts: []int{7},
			wantMin:      1,
		},
	}

	svc := createMockService(t)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			params := data.ArtistFilterParams{MemberCounts: tt.memberCounts}

			results := svc.FilterArtists(params)

			if len(results) < tt.wantMin {
				t.Errorf("FilterArtists() returned %d artists, want at least %d", len(results), tt.wantMin)
				t.Logf("Got artists: %v", getArtistNames(results))
			}

			for _, artist := range results {
				memberCount := len(artist.Members)
				if !containsInt(tt.memberCounts, memberCount) {
					t.Errorf("Artist %s has %d members, not in allowed list %v", artist.Name, memberCount, tt.memberCounts)
				}
			}
		})
	}
}

func TestService_FilterArtists_Countries(t *testing.T) {
	tests := []struct {
		name      string
		countries []string
		wantMin   int
	}{
		{
			name:      "USA concerts only",
			countries: []string{"USA"},
			wantMin:   10,
		},
		{
			name:      "UK concerts only",
			countries: []string{"UK"},
			wantMin:   2,
		},
		{
			name:      "USA or UK concerts",
			countries: []string{"USA", "UK"},
			wantMin:   12,
		},
	}

	svc := createMockService(t)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			params := data.ArtistFilterParams{Countries: tt.countries}

			results := svc.FilterArtists(params)

			if len(results) < tt.wantMin {
				t.Errorf("FilterArtists() returned %d artists, want at least %d", len(results), tt.wantMin)
				t.Logf("Got artists: %v", getArtistNames(results))
			}
		})
	}
}

func TestService_FilterArtists_Combined(t *testing.T) {
	tests := []struct {
		name    string
		params  data.ArtistFilterParams
		wantMin int
	}{
		{
			name: "Solo artists formed 1970-2000",
			params: data.ArtistFilterParams{
				CreationYearFrom: intPtr(1970),
				CreationYearTo:   intPtr(2000),
				MemberCounts:     []int{1},
			},
			wantMin: 0,
		},
		{
			name: "Small bands with USA concerts formed after 2000",
			params: data.ArtistFilterParams{
				CreationYearFrom: intPtr(2000),
				MemberCounts:     []int{2, 3, 4},
				Countries:        []string{"USA"},
			},
			wantMin: 0,
		},
	}

	svc := createMockService(t)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results := svc.FilterArtists(tt.params)

			if len(results) < tt.wantMin {
				t.Errorf("FilterArtists() returned %d artists, want at least %d", len(results), tt.wantMin)
				t.Logf("Got artists: %v", getArtistNames(results))
			}
		})
	}
}

func TestService_GetArtistFilterOptions(t *testing.T) {
	svc := createMockService(t)
	options := svc.GetArtistFilterOptions()

	if options.CreationYearMin == 0 || options.CreationYearMax == 0 {
		t.Error("Creation year bounds not set properly")
	}
	if options.CreationYearMin >= options.CreationYearMax {
		t.Error("Creation year min should be less than max")
	}

	if options.FirstAlbumYearMin == 0 || options.FirstAlbumYearMax == 0 {
		t.Error("First album year bounds not set properly")
	}

	if len(options.MemberCounts) == 0 {
		t.Error("No member counts available")
	}
	if options.MemberCounts[0] != 1 {
		t.Error("Member counts should start from 1")
	}

	if len(options.Countries) == 0 {
		t.Error("No countries available")
	}
	if !contains(options.Countries, "USA") {
		t.Error("Expected USA to be in countries list")
	}
}

func createMockService(t *testing.T) *Service {
	t.Helper()

	mockArtists := []data.Artist{
		{ID: 1, Name: "Queen", Members: []string{"Freddie Mercury", "Brian May", "Roger Taylor", "John Deacon", "Mike Grose", "Barry Mitchell", "Doug Fogie"}, CreationYear: 1970, FirstAlbum: "14-07-1973", Concerts: []data.Concert{{Location: "london-uk"}, {Location: "new-york-usa"}}},
		{ID: 2, Name: "Gorillaz", Members: []string{"Damon Albarn", "Jamie Hewlett"}, CreationYear: 1998, FirstAlbum: "26-03-2001", Concerts: []data.Concert{{Location: "london-uk"}}},
		{ID: 3, Name: "Travis Scott", Members: []string{"Jacques Berman Webster II"}, CreationYear: 2008, FirstAlbum: "2015", Concerts: []data.Concert{{Location: "houston-texas-usa"}, {Location: "atlanta-georgia-usa"}, {Location: "chicago-illinois-usa"}, {Location: "los-angeles-california-usa"}, {Location: "miami-florida-usa"}, {Location: "new-york-usa"}, {Location: "philadelphia-pennsylvania-usa"}, {Location: "phoenix-arizona-usa"}}},
		{ID: 4, Name: "Foo Fighters", Members: []string{"Dave Grohl", "Pat Smear", "Chris Shiflett", "Nate Mendel", "Taylor Hawkins", "Rami Jaffee"}, CreationYear: 1994, FirstAlbum: "04-07-1995", Concerts: []data.Concert{{Location: "seattle-washington-usa"}}},
		{ID: 5, Name: "XXXTentacion", Members: []string{"Jahseh Dwayne Ricardo Onfroy"}, CreationYear: 2013, FirstAlbum: "2017", Concerts: []data.Concert{{Location: "miami-florida-usa"}}},
		{ID: 6, Name: "Juice WRLD", Members: []string{"Jarad Anthony Higgins"}, CreationYear: 2015, FirstAlbum: "2018", Concerts: []data.Concert{{Location: "chicago-illinois-usa"}}},
		{ID: 7, Name: "Alec Benjamin", Members: []string{"Alec Shane Benjamin"}, CreationYear: 2013, FirstAlbum: "2018", Concerts: []data.Concert{{Location: "los-angeles-california-usa"}}},
		{ID: 8, Name: "Post Malone", Members: []string{"Austin Richard Post"}, CreationYear: 2013, FirstAlbum: "2016", Concerts: []data.Concert{{Location: "new-york-usa"}}},
		{ID: 9, Name: "SOJA", Members: []string{"Jacob Hemphill", "Bob Jefferson", "Patrick O'Shea", "Ryan Berty", "Ken Brownell", "Rafael Rodriguez", "Trevor Young", "Hellman Escorcia"}, CreationYear: 1997, FirstAlbum: "2000", Concerts: []data.Concert{{Location: "washington-usa"}}},
		{ID: 10, Name: "Mamonas Assassinas", Members: []string{"Dinho", "Júlio Rasec", "Bento Hinoto", "Sérgio Reoli", "Samuel Reoli"}, CreationYear: 1995, FirstAlbum: "1995", Concerts: []data.Concert{{Location: "sao-paulo-brazil"}}},
		{ID: 11, Name: "Thirty Seconds to Mars", Members: []string{"Jared Leto", "Shannon Leto", "Tomo Miličević"}, CreationYear: 1998, FirstAlbum: "2002", Concerts: []data.Concert{{Location: "los-angeles-california-usa"}}},
		{ID: 12, Name: "Nickelback", Members: []string{"Chad Kroeger", "Ryan Peake", "Mike Kroeger", "Daniel Adair"}, CreationYear: 1995, FirstAlbum: "1996", Concerts: []data.Concert{{Location: "vancouver-canada"}}},
		{ID: 13, Name: "Linkin Park", Members: []string{"Chester Bennington", "Mike Shinoda", "Brad Delson", "Dave Farrell", "Joe Hahn", "Rob Bourdon"}, CreationYear: 1996, FirstAlbum: "2000", Concerts: []data.Concert{{Location: "los-angeles-california-usa"}}},
		{ID: 14, Name: "Coldplay", Members: []string{"Chris Martin", "Guy Berryman", "Jonny Buckland", "Will Champion"}, CreationYear: 1996, FirstAlbum: "2000", Concerts: []data.Concert{{Location: "london-uk"}}},
		{ID: 15, Name: "Red Hot Chili Peppers", Members: []string{"Anthony Kiedis", "Flea", "Chad Smith", "John Frusciante"}, CreationYear: 1982, FirstAlbum: "1991", Concerts: []data.Concert{{Location: "los-angeles-california-usa"}}},
	}

	store := data.NewStoreFromFixtures(mockArtists, nil)
	return New(store)
}

func getArtistNames(artists []data.Artist) []string {
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

func containsInt(slice []int, item int) bool {
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
