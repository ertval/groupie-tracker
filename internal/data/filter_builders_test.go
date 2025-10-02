package data

import "testing"

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
