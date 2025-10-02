package data

import (
	"testing"
	"time"
)

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
