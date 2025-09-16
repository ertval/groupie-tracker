package data

import (
	"context"
	"testing"
)

// TestArtist_Validation tests the validation of Artist structs
func TestArtist_Validation(t *testing.T) {
	tests := []struct {
		name    string
		artist  Artist
		wantErr bool
	}{
		{
			name: "valid artist",
			artist: Artist{
				ID:           1,
				Name:         "Queen",
				Image:        "https://example.com/queen.jpg",
				Members:      []string{"Freddie Mercury", "Brian May"},
				CreationYear: 1970,
				FirstAlbum:   "14-12-1973",
			},
			wantErr: false,
		},
		{
			name: "empty name",
			artist: Artist{
				ID:           1,
				Name:         "",
				Image:        "https://example.com/queen.jpg",
				Members:      []string{"Freddie Mercury"},
				CreationYear: 1970,
				FirstAlbum:   "14-12-1973",
			},
			wantErr: true,
		},
		{
			name: "invalid creation year",
			artist: Artist{
				ID:           1,
				Name:         "Queen",
				Image:        "https://example.com/queen.jpg",
				Members:      []string{"Freddie Mercury"},
				CreationYear: 0,
				FirstAlbum:   "14-12-1973",
			},
			wantErr: true,
		},
		{
			name: "no members",
			artist: Artist{
				ID:           1,
				Name:         "Queen",
				Image:        "https://example.com/queen.jpg",
				Members:      []string{},
				CreationYear: 1970,
				FirstAlbum:   "14-12-1973",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.artist.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Artist.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestArtist_GetFirstAlbumDate tests parsing of first album date
func TestArtist_GetFirstAlbumDate(t *testing.T) {
	tests := []struct {
		name      string
		artist    Artist
		wantYear  int
		wantMonth int
		wantDay   int
		wantErr   bool
	}{
		{
			name: "valid date",
			artist: Artist{
				FirstAlbum: "14-12-1973",
			},
			wantYear:  1973,
			wantMonth: 12,
			wantDay:   14,
			wantErr:   false,
		},
		{
			name: "empty date",
			artist: Artist{
				FirstAlbum: "",
			},
			wantErr: true,
		},
		{
			name: "invalid format",
			artist: Artist{
				FirstAlbum: "1973-12-14",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			date, err := tt.artist.GetFirstAlbumDate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Artist.GetFirstAlbumDate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if date.Year() != tt.wantYear || int(date.Month()) != tt.wantMonth || date.Day() != tt.wantDay {
					t.Errorf("Artist.GetFirstAlbumDate() = %v, want %d-%02d-%02d", date, tt.wantYear, tt.wantMonth, tt.wantDay)
				}
			}
		})
	}
}

// TestArtist_GenerateSlug tests slug generation
func TestArtist_GenerateSlug(t *testing.T) {
	tests := []struct {
		name     string
		artist   Artist
		expected string
	}{
		{
			name:     "simple name",
			artist:   Artist{Name: "Queen"},
			expected: "queen",
		},
		{
			name:     "name with spaces",
			artist:   Artist{Name: "Pink Floyd"},
			expected: "pink-floyd",
		},
		{
			name:     "name with special characters",
			artist:   Artist{Name: "AC/DC"},
			expected: "ac-dc",
		},
		{
			name:     "name with multiple spaces",
			artist:   Artist{Name: "The   Beatles"},
			expected: "the-beatles",
		},
		{
			name:     "empty name",
			artist:   Artist{Name: ""},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.artist.GenerateSlug()
			if result != tt.expected {
				t.Errorf("Artist.GenerateSlug() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// TestArtist_SetAndGetSlug tests slug setting and getting
func TestArtist_SetAndGetSlug(t *testing.T) {
	tests := []struct {
		name     string
		artist   Artist
		expected string
	}{
		{
			name:     "set slug for Queen",
			artist:   Artist{Name: "Queen"},
			expected: "queen",
		},
		{
			name:     "set slug for Pink Floyd",
			artist:   Artist{Name: "Pink Floyd"},
			expected: "pink-floyd",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.artist.SetSlug()
			if tt.artist.GetSlug() != tt.expected {
				t.Errorf("Artist.GetSlug() after SetSlug() = %v, want %v", tt.artist.GetSlug(), tt.expected)
			}
		})
	}
}

// TestRelation_Validation tests Relation validation
func TestRelation_Validation(t *testing.T) {
	tests := []struct {
		name     string
		relation Relation
		wantErr  bool
	}{
		{
			name: "valid relation",
			relation: Relation{
				ID: 1,
				DatesLocations: map[string][]string{
					"new_york-usa": {"01-01-2020", "02-01-2020"},
					"london-uk":    {"03-01-2020"},
				},
			},
			wantErr: false,
		},
		{
			name: "invalid ID",
			relation: Relation{
				ID:             0,
				DatesLocations: map[string][]string{"new_york-usa": {"01-01-2020"}},
			},
			wantErr: true,
		},
		{
			name: "no dates locations",
			relation: Relation{
				ID:             1,
				DatesLocations: map[string][]string{},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.relation.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Relation.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
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

// MockAPIClient for testing
type MockAPIClient struct {
	data *APIResponse
	err  error
}

func (m *MockAPIClient) FetchAllData(ctx context.Context) (*APIResponse, error) {
	return m.data, m.err
}

// TestRepository_InitializeWithAPI tests repository initialization with API client
func TestRepository_InitializeWithAPI(t *testing.T) {
	t.Run("successful initialization", func(t *testing.T) {
		mockData := &APIResponse{
			Artists: []Artist{
				{ID: 1, Name: "Queen", Members: []string{"Freddie Mercury"}, CreationYear: 1970, FirstAlbum: "14-12-1973"},
				{ID: 2, Name: "Pink Floyd", Members: []string{"David Gilmour"}, CreationYear: 1965, FirstAlbum: "05-08-1967"},
			},
			Relations: []Relation{
				{ID: 1, DatesLocations: map[string][]string{"new_york-usa": {"01-01-2020"}}},
				{ID: 2, DatesLocations: map[string][]string{"london-uk": {"02-01-2020"}}},
			},
		}

		repo := NewRepository()
		mockClient := &MockAPIClient{data: mockData, err: nil}

		ctx := context.Background()
		err := repo.InitializeWithAPI(ctx, mockClient)

		if err != nil {
			t.Errorf("Repository.InitializeWithAPI() error = %v, want nil", err)
		}

		artists := repo.GetAllArtists()
		if len(artists) != 2 {
			t.Errorf("Expected 2 artists, got %d", len(artists))
		}

		relations := repo.GetAllRelations()
		if len(relations) != 2 {
			t.Errorf("Expected 2 relations, got %d", len(relations))
		}
	})

	t.Run("API client error", func(t *testing.T) {
		repo := NewRepository()
		mockClient := &MockAPIClient{data: nil, err: context.DeadlineExceeded}

		ctx := context.Background()
		err := repo.InitializeWithAPI(ctx, mockClient)

		if err == nil {
			t.Errorf("Expected error from API client, got nil")
		}
	})
}

// TestRepository_GetArtistBySlug tests retrieving artists by slug
func TestRepository_GetArtistBySlug(t *testing.T) {
	repo := NewRepository()

	// Load test data
	testData := APIResponse{
		Artists: []Artist{
			{ID: 1, Name: "Queen", Members: []string{"Freddie Mercury"}, CreationYear: 1970, FirstAlbum: "14-12-1973"},
			{ID: 2, Name: "Pink Floyd", Members: []string{"David Gilmour"}, CreationYear: 1965, FirstAlbum: "05-08-1967"},
		},
	}
	repo.LoadData(testData)

	t.Run("existing slug", func(t *testing.T) {
		artist, found := repo.GetArtistBySlug("queen")
		if !found {
			t.Errorf("Expected to find artist with slug 'queen'")
		}
		if artist.Name != "Queen" {
			t.Errorf("Expected Queen, got %s", artist.Name)
		}
	})

	t.Run("non-existing slug", func(t *testing.T) {
		_, found := repo.GetArtistBySlug("nonexistent")
		if found {
			t.Errorf("Expected not to find artist with slug 'nonexistent'")
		}
	})
}

// TestRepository_CalculateLocationStats tests location statistics calculation
func TestRepository_CalculateLocationStats(t *testing.T) {
	repo := NewRepository()

	// Load test data
	testData := APIResponse{
		Artists: []Artist{
			{ID: 1, Name: "Queen", Members: []string{"Freddie Mercury"}, CreationYear: 1970, FirstAlbum: "14-12-1973"},
			{ID: 2, Name: "Pink Floyd", Members: []string{"David Gilmour"}, CreationYear: 1965, FirstAlbum: "05-08-1967"},
		},
		Relations: []Relation{
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
	repo.LoadData(testData)

	stats := repo.CalculateLocationStats()

	if len(stats) != 3 {
		t.Errorf("Expected 3 location stats, got %d", len(stats))
	}

	// Check if sorted by concert count (descending)
	if len(stats) > 1 && stats[0].ConcertCount < stats[1].ConcertCount {
		t.Errorf("Location stats not sorted by concert count")
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
		}
	}
	if !newYorkFound {
		t.Errorf("New York location not found in stats")
	}
}

// TestRepository_GetStats tests comprehensive statistics calculation
func TestRepository_GetStats(t *testing.T) {
	repo := NewRepository()

	// Load test data
	testData := APIResponse{
		Artists: []Artist{
			{ID: 1, Name: "Queen", Members: []string{"Freddie Mercury"}, CreationYear: 1970, FirstAlbum: "14-12-1973"},
			{ID: 2, Name: "Pink Floyd", Members: []string{"David Gilmour"}, CreationYear: 1965, FirstAlbum: "05-08-1967"},
		},
		Relations: []Relation{
			{ID: 1, DatesLocations: map[string][]string{
				"new_york-usa": {"01-01-2020", "02-01-2020"},
				"london-uk":    {"03-01-2020"},
			}},
			{ID: 2, DatesLocations: map[string][]string{
				"new_york-usa": {"04-01-2020"},
			}},
		},
	}
	repo.LoadData(testData)

	stats := repo.GetStats()

	if stats["artists"] != 2 {
		t.Errorf("Expected 2 artists, got %d", stats["artists"])
	}

	if stats["relations"] != 2 {
		t.Errorf("Expected 2 relations, got %d", stats["relations"])
	}

	if stats["total_concerts"] != 4 {
		t.Errorf("Expected 4 total concerts, got %d", stats["total_concerts"])
	}

	if stats["locations"] != 2 {
		t.Errorf("Expected 2 unique locations, got %d", stats["locations"])
	}
}

// TestRepository_GetAllArtistsSorted tests sorted artist retrieval
func TestRepository_GetAllArtistsSorted(t *testing.T) {
	repo := NewRepository()

	// Load test data
	testData := APIResponse{
		Artists: []Artist{
			{ID: 1, Name: "Queen", Members: []string{"Freddie Mercury"}, CreationYear: 1970, FirstAlbum: "14-12-1973"},
			{ID: 2, Name: "AC/DC", Members: []string{"Angus Young"}, CreationYear: 1973, FirstAlbum: "17-02-1975"},
			{ID: 3, Name: "Pink Floyd", Members: []string{"David Gilmour"}, CreationYear: 1965, FirstAlbum: "05-08-1967"},
		},
	}
	repo.LoadData(testData)

	artists := repo.GetAllArtistsSorted()

	if len(artists) != 3 {
		t.Errorf("Expected 3 artists, got %d", len(artists))
	}

	// Check alphabetical order (case-insensitive)
	if artists[0].Name != "AC/DC" || artists[1].Name != "Pink Floyd" || artists[2].Name != "Queen" {
		t.Errorf("Artists not sorted alphabetically: got %v, %v, %v", artists[0].Name, artists[1].Name, artists[2].Name)
	}
}

// TestRepository_GetArtistNavigation tests artist navigation functionality
func TestRepository_GetArtistNavigation(t *testing.T) {
	repo := NewRepository()

	// Load test data
	testData := APIResponse{
		Artists: []Artist{
			{ID: 1, Name: "Queen", Members: []string{"Freddie Mercury"}, CreationYear: 1970, FirstAlbum: "14-12-1973"},
			{ID: 2, Name: "AC/DC", Members: []string{"Angus Young"}, CreationYear: 1973, FirstAlbum: "17-02-1975"},
			{ID: 3, Name: "Pink Floyd", Members: []string{"David Gilmour"}, CreationYear: 1965, FirstAlbum: "05-08-1967"},
		},
	}
	repo.LoadData(testData)

	queen, _ := repo.GetArtistBySlug("queen")
	prev, next := repo.GetArtistNavigation(queen)

	// Queen should be the last artist alphabetically
	if prev == nil || prev.Name != "Pink Floyd" {
		t.Errorf("Expected previous artist to be Pink Floyd, got %v", prev)
	}

	if next != nil {
		t.Errorf("Expected next artist to be nil, got %v", next)
	}
}
