package data

import (
	"testing"
)

// TestRepository_NewRepository tests repository creation
func TestRepository_NewRepository(t *testing.T) {
	repo := NewRepository()

	if repo == nil {
		t.Error("NewRepository() returned nil")
	}

	if repo.artists == nil || repo.relations == nil || repo.artistSlugs == nil {
		t.Error("Repository maps not properly initialized")
	}

	if len(repo.GetAllArtists()) != 0 {
		t.Error("New repository should have no artists")
	}

	if len(repo.GetAllRelations()) != 0 {
		t.Error("New repository should have no relations")
	}
}

// TestRepository_LoadData tests loading API data into repository
func TestRepository_LoadData(t *testing.T) {
	repo := NewRepository()

	apiData := APIResponse{
		Artists: []APIArtist{
			{ID: 1, Name: "Queen", Members: []string{"Freddie Mercury"}, CreationYear: 1970, FirstAlbum: "14-12-1973", Image: "queen.jpg"},
			{ID: 2, Name: "Pink Floyd", Members: []string{"David Gilmour"}, CreationYear: 1965, FirstAlbum: "05-08-1967", Image: "pinkfloyd.jpg"},
		},
		Relations: []APIRelation{
			{ID: 1, DatesLocations: map[string][]string{"new_york-usa": {"01-01-2020"}}},
			{ID: 2, DatesLocations: map[string][]string{"london-uk": {"02-01-2020"}}},
		},
	}

	repo.LoadData(apiData)

	// Check artists were loaded
	artists := repo.GetAllArtists()
	if len(artists) != 2 {
		t.Errorf("Expected 2 artists, got %d", len(artists))
	}

	// Check relations were loaded
	relations := repo.GetAllRelations()
	if len(relations) != 2 {
		t.Errorf("Expected 2 relations, got %d", len(relations))
	}

	// Check artist by ID
	artist, found := repo.GetArtist(1)
	if !found {
		t.Error("Expected to find artist with ID 1")
	}
	if artist.Name != "Queen" {
		t.Errorf("Expected Queen, got %s", artist.Name)
	}

	// Check artist by slug
	artist, found = repo.GetArtistBySlug("queen")
	if !found {
		t.Error("Expected to find artist with slug 'queen'")
	}
	if artist.Name != "Queen" {
		t.Errorf("Expected Queen, got %s", artist.Name)
	}
}

// TestRepository_GetAllArtistsSorted tests sorted artist retrieval
func TestRepository_GetAllArtistsSorted(t *testing.T) {
	repo := NewRepository()

	apiData := APIResponse{
		Artists: []APIArtist{
			{ID: 1, Name: "Queen", Members: []string{"Freddie Mercury"}, CreationYear: 1970, FirstAlbum: "14-12-1973"},
			{ID: 2, Name: "AC/DC", Members: []string{"Angus Young"}, CreationYear: 1973, FirstAlbum: "17-02-1975"},
			{ID: 3, Name: "Pink Floyd", Members: []string{"David Gilmour"}, CreationYear: 1965, FirstAlbum: "05-08-1967"},
		},
	}
	repo.LoadData(apiData)

	artists := repo.GetAllArtistsSorted()

	if len(artists) != 3 {
		t.Errorf("Expected 3 artists, got %d", len(artists))
	}

	// Check alphabetical order
	expectedOrder := []string{"AC/DC", "Pink Floyd", "Queen"}
	for i, artist := range artists {
		if artist.Name != expectedOrder[i] {
			t.Errorf("Artists not sorted alphabetically at position %d: got %v, want %v", i, artist.Name, expectedOrder[i])
		}
	}
}

// TestRepository_CalculateLocationStats tests location statistics calculation
func TestRepository_CalculateLocationStats(t *testing.T) {
	repo := NewRepository()

	apiData := APIResponse{
		Artists: []APIArtist{
			{ID: 1, Name: "Queen", Members: []string{"Freddie Mercury"}, CreationYear: 1970, FirstAlbum: "14-12-1973"},
			{ID: 2, Name: "Pink Floyd", Members: []string{"David Gilmour"}, CreationYear: 1965, FirstAlbum: "05-08-1967"},
		},
		Relations: []APIRelation{
			{ID: 1, DatesLocations: map[string][]string{
				"new_york-usa": {"01-01-2020", "02-01-2020"},
				"london-uk":    {"03-01-2020"},
			}},
			{ID: 2, DatesLocations: map[string][]string{
				"new_york-usa": {"04-01-2020"},
				"paris-france": {"05-01-2020"},
			}},
		},
	}
	repo.LoadData(apiData)

	stats := repo.CalculateLocationStats()

	if len(stats) != 3 {
		t.Errorf("Expected 3 location stats, got %d", len(stats))
	}

	// Check if sorted by artist count (descending)
	if len(stats) > 1 && stats[0].ArtistCount < stats[1].ArtistCount {
		t.Errorf("Location stats not sorted by artist count")
	}

	// Check specific location stats
	newYorkFound := false
	for _, stat := range stats {
		if stat.Name == "new_york-usa" {
			newYorkFound = true
			if stat.ArtistCount != 2 {
				t.Errorf("Expected New York to have 2 artists, got %d", stat.ArtistCount)
			}
			if stat.ConcertCount != 3 {
				t.Errorf("Expected New York to have 3 concerts, got %d", stat.ConcertCount)
			}
			if stat.DisplayName != "New York, USA" {
				t.Errorf("Expected display name 'New York, USA', got %s", stat.DisplayName)
			}
		}
	}
	if !newYorkFound {
		t.Errorf("New York location not found in stats")
	}
}

// TestGenerateLocationSlug tests location slug generation
func TestGenerateLocationSlug(t *testing.T) {
	tests := []struct {
		name         string
		locationName string
		expected     string
	}{
		{
			name:         "new york usa",
			locationName: "new_york-usa",
			expected:     "new-york-usa",
		},
		{
			name:         "london uk",
			locationName: "london-uk",
			expected:     "london-uk",
		},
		{
			name:         "empty location",
			locationName: "",
			expected:     "",
		},
		{
			name:         "location with special chars",
			locationName: "san_josé-costa_rica",
			expected:     "san-jos-costa-rica",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GenerateLocationSlug(tt.locationName)
			if result != tt.expected {
				t.Errorf("GenerateLocationSlug() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// TestNormalizeLocationName tests location name normalization
func TestNormalizeLocationName(t *testing.T) {
	tests := []struct {
		name         string
		locationName string
		expected     string
	}{
		{
			name:         "new york usa",
			locationName: "new_york-usa",
			expected:     "New York, USA",
		},
		{
			name:         "london uk",
			locationName: "london-uk",
			expected:     "London, UK",
		},
		{
			name:         "empty location",
			locationName: "",
			expected:     "",
		},
		{
			name:         "single word",
			locationName: "berlin",
			expected:     "Berlin",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := NormalizeLocationName(tt.locationName)
			if result != tt.expected {
				t.Errorf("NormalizeLocationName() = %v, want %v", result, tt.expected)
			}
		})
	}
}
