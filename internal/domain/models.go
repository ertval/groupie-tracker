package domain

// Artist represents the complete internal model of a music artist.
type Artist struct {
	ID              int
	Name            string
	Slug            string // URL-friendly identifier (e.g., "queen")
	Members         []string
	CreationYear    int
	FirstAlbum      string
	Image           string
	Concerts        []Concert
	DatesAtLocation map[string][]string // Concert dates indexed by location slug
	ConcertCount    int                 // Computed field
	Countries       []string            // Unique countries where artist performed
}

// ArtistAtLocation represents an artist's concert activity at a specific venue.
type ArtistAtLocation struct {
	Artist       Artist
	ConcertCount int
}

// Location represents the complete internal model of a concert venue.
type Location struct {
	Name          string
	Slug          string // URL-friendly identifier (e.g., "london-uk")
	Artists       []ArtistAtLocation
	ArtistCount   int // Computed field
	TotalConcerts int // Computed field
	EarliestYear  int
	LatestYear    int
}

// Concert represents a single concert event in structured form.
//
// This simplified structure is created by parsing the complex API relations data.
// Each Concert represents one performance at one venue on one date, making it
// easy to filter, count, and analyze concert data across the application.
type Concert struct {
	Date     string // Concert date in original API format
	Location string // Normalized location name matching Location.Name
}

// --- Filter Data Structures ---
//
// These models define the comprehensive filtering system that allows users to narrow
// artist and location results based on multiple criteria. The system uses a combination
// of range-based filters (with sliders) and discrete choice filters (with checkboxes).

// ArtistFilterParams represents all possible filter criteria that can be applied to artist searches.
//
// Range Filters:
//   - Use pointer types (*int) to distinguish between "unset" and "zero" values
//   - Enable inclusive range filtering with optional min/max bounds
//   - Support dual-range sliders in the user interface
//
// Checkbox Filters:
//   - Use slice types ([]int, []string) for multi-selection
//   - Empty slices mean "no filtering applied" for that criterion
//   - Allow users to select multiple values with OR logic within each filter
//
// All filter types use AND logic between different filter categories.
type ArtistFilterParams struct {
	// Range filters using dual-range slider UI components
	CreationYearFrom *int `json:"creationYearFrom,omitempty"` // Minimum band formation year (inclusive)
	CreationYearTo   *int `json:"creationYearTo,omitempty"`   // Maximum band formation year (inclusive)

	FirstAlbumYearFrom *int `json:"firstAlbumYearFrom,omitempty"` // Minimum first album year (inclusive)
	FirstAlbumYearTo   *int `json:"firstAlbumYearTo,omitempty"`   // Maximum first album year (inclusive)

	// Multi-select checkbox filters
	MemberCounts []int    `json:"memberCounts,omitempty"` // Allowed band member counts (exact match)
	Countries    []string `json:"countries,omitempty"`    // Countries where artist must have performed
}

// ArtistFilterOptions represents the complete set of available filter options for artists.
//
// ArtistFilterOptions represents available filter options computed from the dataset.
type ArtistFilterOptions struct {
	CreationYearMin   int      `json:"creationYearMin"`
	CreationYearMax   int      `json:"creationYearMax"`
	FirstAlbumYearMin int      `json:"firstAlbumYearMin"`
	FirstAlbumYearMax int      `json:"firstAlbumYearMax"`
	MemberCounts      []int    `json:"memberCounts"`
	Countries         []string `json:"countries"`
}

// LocationFilterParams represents filter criteria for location searches.
type LocationFilterParams struct {
	ConcertCountFrom *int     `json:"concertCountFrom,omitempty"`
	ConcertCountTo   *int     `json:"concertCountTo,omitempty"`
	ArtistCountFrom  *int     `json:"artistCountFrom,omitempty"`
	ArtistCountTo    *int     `json:"artistCountTo,omitempty"`
	ConcertYearFrom  *int     `json:"concertYearFrom,omitempty"`
	ConcertYearTo    *int     `json:"concertYearTo,omitempty"`
	Countries        []string `json:"countries,omitempty"`
}

// LocationFilterOptions represents available filter options for locations.
type LocationFilterOptions struct {
	ConcertCountMin int      `json:"concertCountMin"`
	ConcertCountMax int      `json:"concertCountMax"`
	ArtistCountMin  int      `json:"artistCountMin"`
	ArtistCountMax  int      `json:"artistCountMax"`
	ConcertYearMin  int      `json:"concertYearMin"`
	ConcertYearMax  int      `json:"concertYearMax"`
	Countries       []string `json:"countries"`
}

// SearchSuggestionType represents the type of search result for categorization.
type SearchSuggestionType string

const (
	SuggestionTypeArtist     SearchSuggestionType = "artist"
	SuggestionTypeMember     SearchSuggestionType = "member"
	SuggestionTypeLocation   SearchSuggestionType = "location"
	SuggestionTypeFirstAlbum SearchSuggestionType = "first-album"
	SuggestionTypeCreation   SearchSuggestionType = "creation"
)

// SearchSuggestion represents a single search suggestion with type identification.
type SearchSuggestion struct {
	Text           string               `json:"text"`
	Type           SearchSuggestionType `json:"type"`
	Description    string               `json:"description"`
	URL            string               `json:"url"`
	ArtistID       int                  `json:"artistId"`
	normalizedText string               `json:"-"` // Lowercase for efficient matching
}

// SearchResult represents a comprehensive search result with matched items.
type SearchResult struct {
	Artists      []Artist `json:"artists"`
	Query        string   `json:"query"`
	TotalResults int      `json:"totalResults"`
}

// SearchParams represents search query parameters from user input.
//
// Supports integration with existing filter system by combining text search
// with advanced filtering options for more precise results.
type SearchParams struct {
	Query   string             `json:"query"`   // Search text input
	Filters ArtistFilterParams `json:"filters"` // Optional additional filters
}

// AppStats represents application-wide statistics with type-safe fields.
//
// This structure provides a type-safe alternative to the map[string]int approach,
// enabling compile-time validation and better API documentation. All fields
// represent counts computed during data loading or runtime.
type AppStats struct {
	TotalArtists     int `json:"total_artists"`     // Number of artists in the dataset
	TotalMembers     int `json:"total_members"`     // Sum of all band members across all artists
	TotalLocations   int `json:"total_locations"`   // Number of unique concert venues
	TotalConcerts    int `json:"total_concerts"`    // Total number of concert events
	TotalCountries   int `json:"total_countries"`   // Number of unique countries with concerts
	CachedImages     int `json:"cached_images"`     // Number of artist images served from local cache
	DownloadedImages int `json:"downloaded_images"` // Number of artist images downloaded this session
}
