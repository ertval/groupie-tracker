package data

import (
	"testing"
	"time"
)

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
