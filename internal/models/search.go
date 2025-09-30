package models

// SearchParams combines search query with optional filters for comprehensive searching.
type SearchParams struct {
	// Query is the main search term to look for across all searchable fields
	Query string `form:"q" json:"query"`

	// Filters contains additional filtering criteria
	Filters Filters `form:"filters" json:"filters"`
}

// SearchResult contains search results and metadata for user feedback.
type SearchResult struct {
	// Artists contains the matching artists from the search
	Artists []Artist `json:"artists"`

	// TotalResults is the count of matching artists
	TotalResults int `json:"total_results"`

	// Query is the original search query for reference
	Query string `json:"query"`
}

// SearchSuggestionType categorizes different types of search suggestions for UI grouping.
type SearchSuggestionType string

const (
	SuggestionTypeArtist     SearchSuggestionType = "artist"
	SuggestionTypeMember     SearchSuggestionType = "member"
	SuggestionTypeLocation   SearchSuggestionType = "location"
	SuggestionTypeFirstAlbum SearchSuggestionType = "first-album"
	SuggestionTypeCreation   SearchSuggestionType = "creation"
)

// SearchSuggestion represents a search autocomplete suggestion with metadata.
type SearchSuggestion struct {
	// Text is the suggestion text to display and use for searching
	Text string `json:"text"`

	// Type categorizes the suggestion for UI grouping
	Type SearchSuggestionType `json:"type"`

	// Description provides additional context about the suggestion
	Description string `json:"description"`

	// URL provides a direct link to the suggested item (optional)
	URL string `json:"url"`

	// ArtistID links the suggestion to a specific artist (when applicable)
	ArtistID int `json:"artist_id,omitempty"`
}

// Filters represents filter parameters with simple zero values instead of pointers.
// Zero values indicate "no filter applied" for that criterion.
type Filters struct {
	// CreationYearMin is the minimum creation year (0 means no minimum)
	CreationYearMin int `form:"creation_year_min" json:"creation_year_min"`

	// CreationYearMax is the maximum creation year (0 means no maximum)
	CreationYearMax int `form:"creation_year_max" json:"creation_year_max"`

	// FirstAlbumYearMin is the minimum first album year (0 means no minimum)
	FirstAlbumYearMin int `form:"first_album_year_min" json:"first_album_year_min"`

	// FirstAlbumYearMax is the maximum first album year (0 means no maximum)
	FirstAlbumYearMax int `form:"first_album_year_max" json:"first_album_year_max"`

	// MemberCounts filters by exact member counts (empty means no filter)
	MemberCounts []int `form:"member_counts" json:"member_counts"`

	// Countries filters by performance countries (empty means no filter)
	Countries []string `form:"countries" json:"countries"`
}

// IsEmpty returns true if no filters are applied.
func (f Filters) IsEmpty() bool {
	return f.CreationYearMin == 0 &&
		f.CreationYearMax == 0 &&
		f.FirstAlbumYearMin == 0 &&
		f.FirstAlbumYearMax == 0 &&
		len(f.MemberCounts) == 0 &&
		len(f.Countries) == 0
}

// FilterOptions provides the available filter bounds and options computed from the dataset.
type FilterOptions struct {
	// CreationYearMin is the earliest creation year in the dataset
	CreationYearMin int `json:"creation_year_min"`

	// CreationYearMax is the latest creation year in the dataset
	CreationYearMax int `json:"creation_year_max"`

	// FirstAlbumYearMin is the earliest first album year in the dataset
	FirstAlbumYearMin int `json:"first_album_year_min"`

	// FirstAlbumYearMax is the latest first album year in the dataset
	FirstAlbumYearMax int `json:"first_album_year_max"`

	// MemberCounts contains all unique member counts found in the dataset
	MemberCounts []int `json:"member_counts"`

	// Countries contains all unique countries found in the dataset
	Countries []string `json:"countries"`
}
