package testsupport

import (
	"context"

	"groupie-tracker/internal/api"
)

// StubLoader satisfies the data.Loader interface for tests with deterministic fixtures.
type StubLoader struct {
	Artists   []api.Artist
	Relations []api.RelationIndex
}

// FetchArtists returns a defensive copy of the stubbed artist slice.
func (s StubLoader) FetchArtists(ctx context.Context) ([]api.Artist, error) {
	artists := make([]api.Artist, len(s.Artists))
	copy(artists, s.Artists)
	return artists, nil
}

// FetchRelations returns a defensive copy of the stubbed relation slice.
func (s StubLoader) FetchRelations(ctx context.Context) ([]api.RelationIndex, error) {
	relations := make([]api.RelationIndex, len(s.Relations))
	copy(relations, s.Relations)
	return relations, nil
}

// MinimalDataset provides a compact but representative dataset used across tests.
func MinimalDataset() StubLoader {
	artists := []api.Artist{
		{
			ID:           1,
			Name:         "The Example",
			Members:      []string{"Alice", "Bob"},
			CreationYear: 2000,
			FirstAlbum:   "2001-01-01",
			Image:        "example.jpg",
		},
		{
			ID:           2,
			Name:         "Solo Artist",
			Members:      []string{"Charlie"},
			CreationYear: 2010,
			FirstAlbum:   "2012",
			Image:        "solo.jpg",
		},
	}

	relations := []api.RelationIndex{
		{
			ID: 1,
			DatesLocations: map[string][]string{
				"new-york-usa": {"2005-03-12"},
				"london-uk":    {"2006-07-18"},
			},
		},
		{
			ID: 2,
			DatesLocations: map[string][]string{
				"sao-paulo-brazil": {"2014-05-05", "2015-05-05"},
			},
		},
	}

	return StubLoader{Artists: artists, Relations: relations}
}
