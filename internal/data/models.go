package data

// --- Core Domain Models ---
// Simplified domain models with essential fields only

// Artist represents a music artist/band with computed performance data.
type Artist struct {
	ID           int
	Name         string
	Slug         string
	Members      []string
	CreationYear int
	FirstAlbum   string
	Image        string
	Concerts     []Concert
	Countries    []string
	ConcertCount int
}

// Location represents a concert venue with aggregated statistics.
type Location struct {
	Name         string
	Slug         string
	Artists      []ArtistSummary
	ConcertCount int
	YearRange    [2]int // [earliest, latest]
}

// Concert represents a single concert event.
type Concert struct {
	Date     string
	Location string
}

// ArtistSummary represents basic artist info for location displays.
type ArtistSummary struct {
	Name         string
	Slug         string
	ConcertCount int
}

// --- Filter Structures ---

// ArtistFilterParams defines filter criteria for artist searches.
type ArtistFilterParams struct {
	CreationYearFrom *int     `json:"creationYearFrom,omitempty"`
	CreationYearTo   *int     `json:"creationYearTo,omitempty"`
	MemberCounts     []int    `json:"memberCounts,omitempty"`
	Countries        []string `json:"countries,omitempty"`
}

// ArtistFilterOptions defines available filter options.
type ArtistFilterOptions struct {
	CreationYearMin int
	CreationYearMax int
	MemberCounts    []int
	Countries       []string
}

// LocationFilterParams defines filter criteria for location searches.
type LocationFilterParams struct {
	YearFrom     *int  `json:"yearFrom,omitempty"`
	YearTo       *int  `json:"yearTo,omitempty"`
	ArtistCounts []int `json:"artistCounts,omitempty"`
}

// LocationFilterOptions defines available location filter options.
type LocationFilterOptions struct {
	YearMin      int
	YearMax      int
	ArtistCounts []int
}

// SearchSuggestion represents a search autocomplete suggestion.
type SearchSuggestion struct {
	Text     string `json:"text"`
	Type     string `json:"type"`
	Category string `json:"category"`
}

// Stats holds application-wide statistics.
type Stats struct {
	TotalArtists   int
	TotalLocations int
	TotalMembers   int
	TotalConcerts  int
	TotalCountries int
}

// SearchParams defines parameters for search operations.
type SearchParams struct {
	Query string `json:"query"`
	Type  string `json:"type"` // "all", "artists", "locations"
	Limit int    `json:"limit"`
}

// SearchResults contains search results for different entity types.
type SearchResults struct {
	Artists   []Artist   `json:"artists"`
	Locations []Location `json:"locations"`
	Total     int        `json:"total"`
}

// Legacy types for compatibility
type AppStats = Stats
type SearchResult = SearchResults
