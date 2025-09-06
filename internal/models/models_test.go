package models

import (
	"testing"
	"time"
)

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

func TestArtist_GetFirstAlbumDate(t *testing.T) {
	tests := []struct {
		name       string
		firstAlbum string
		want       time.Time
		wantErr    bool
	}{
		{
			name:       "valid date format",
			firstAlbum: "26-03-2001",
			want:       time.Date(2001, 3, 26, 0, 0, 0, 0, time.UTC),
			wantErr:    false,
		},
		{
			name:       "invalid date format",
			firstAlbum: "2001-03-26",
			want:       time.Time{},
			wantErr:    true,
		},
		{
			name:       "empty date",
			firstAlbum: "",
			want:       time.Time{},
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			artist := Artist{FirstAlbum: tt.firstAlbum}
			got, err := artist.GetFirstAlbumDate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Artist.GetFirstAlbumDate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !got.Equal(tt.want) {
				t.Errorf("Artist.GetFirstAlbumDate() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestLocation_Validation(t *testing.T) {
	tests := []struct {
		name     string
		location Location
		wantErr  bool
	}{
		{
			name: "valid location",
			location: Location{
				ID:        1,
				Locations: []string{"new_york-usa", "london-uk"},
			},
			wantErr: false,
		},
		{
			name: "empty locations",
			location: Location{
				ID:        1,
				Locations: []string{},
			},
			wantErr: true,
		},
		{
			name: "invalid ID",
			location: Location{
				ID:        0,
				Locations: []string{"new_york-usa"},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.location.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Location.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestDate_Validation(t *testing.T) {
	tests := []struct {
		name    string
		date    Date
		wantErr bool
	}{
		{
			name: "valid date",
			date: Date{
				ID:    1,
				Dates: []string{"23-08-2019", "22-08-2019"},
			},
			wantErr: false,
		},
		{
			name: "empty dates",
			date: Date{
				ID:    1,
				Dates: []string{},
			},
			wantErr: true,
		},
		{
			name: "invalid ID",
			date: Date{
				ID:    0,
				Dates: []string{"23-08-2019"},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.date.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Date.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

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
					"london-uk":    {"23-08-2019", "22-08-2019"},
					"new_york-usa": {"25-08-2019"},
				},
			},
			wantErr: false,
		},
		{
			name: "empty dates locations",
			relation: Relation{
				ID:             1,
				DatesLocations: map[string][]string{},
			},
			wantErr: true,
		},
		{
			name: "invalid ID",
			relation: Relation{
				ID: 0,
				DatesLocations: map[string][]string{
					"london-uk": {"23-08-2019"},
				},
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
