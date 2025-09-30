package search

import "groupie-tracker/internal/data"

// Filters represents the filter parameters that can be applied to artist searches.
type Filters struct {
	CreationYearMin   int      `json:"creation_year_min"`
	CreationYearMax   int      `json:"creation_year_max"`
	FirstAlbumYearMin int      `json:"first_album_year_min"`
	FirstAlbumYearMax int      `json:"first_album_year_max"`
	MemberCounts      []int    `json:"member_counts"`
	Countries         []string `json:"countries"`
}

// FilterOptions describes the available ranges and values for the filter UI.
type FilterOptions struct {
	CreationYearMin   int      `json:"creation_year_min"`
	CreationYearMax   int      `json:"creation_year_max"`
	FirstAlbumYearMin int      `json:"first_album_year_min"`
	FirstAlbumYearMax int      `json:"first_album_year_max"`
	MemberCounts      []int    `json:"member_counts"`
	Countries         []string `json:"countries"`
}

// Params combines a free-text query with optional filters.
type Params struct {
	Query   string  `json:"query"`
	Filters Filters `json:"filters"`
}

// Result represents the outcome of a search operation.
type Result struct {
	Artists      []data.Artist `json:"artists"`
	TotalResults int           `json:"total_results"`
	Query        string        `json:"query"`
}

// SuggestionType categorises suggestions shown in autocomplete.
type SuggestionType string

// Suggestion type values.
const (
	SuggestionArtist     SuggestionType = "artist"
	SuggestionMember     SuggestionType = "member"
	SuggestionLocation   SuggestionType = "location"
	SuggestionFirstAlbum SuggestionType = "first-album"
	SuggestionCreation   SuggestionType = "creation"
)

// Suggestion is a single autocomplete entry.
type Suggestion struct {
	Text        string         `json:"text"`
	Type        SuggestionType `json:"type"`
	Description string         `json:"description"`
	URL         string         `json:"url"`
	ArtistID    int            `json:"artist_id,omitempty"`
}

// Service exposes search, filter and suggestion helpers backed by the data store.
type Service struct {
	provider    ArtistProvider
	suggestions []Suggestion
}

// ArtistProvider is the subset of store behaviour required by the search service.
type ArtistProvider interface {
	Artists() []data.Artist
}
