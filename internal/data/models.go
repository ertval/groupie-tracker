package data

// --- External API Data Structures ---
//
// These models represent the exact JSON structure returned by the Groupie Tracker API.
// They are used only for initial data loading and are converted to rich domain models
// immediately after parsing. Field names and types match the API specification exactly.

// APIArtist represents the raw artist data structure from the /api/artists endpoint.
//
// This is a direct mapping of the external API response with minimal processing.
// The CreationYear field is mapped from the API's "creationDate" for consistency.
// These records are converted to the richer Artist domain model after loading.
type APIArtist struct {
	ID           int      `json:"id"`           // Unique artist identifier from API
	Name         string   `json:"name"`         // Artist/band name as provided by API
	Members      []string `json:"members"`      // Current band member names
	CreationYear int      `json:"creationDate"` // Band formation year (note JSON name mapping)
	FirstAlbum   string   `json:"firstAlbum"`   // First album release date string
	Image        string   `json:"image"`        // Artist image URL from API
}

// APIRelationIndex represents a single artist's concert data from the /api/relation endpoint.
//
// The DatesLocations map structure directly reflects the API format where:
//   - Keys are location strings (e.g., "london-uk", "new-york-usa")
//   - Values are arrays of date strings for concerts at that location
//
// This nested structure is flattened into Concert objects during processing.
type APIRelationIndex struct {
	ID             int                 `json:"id"`             // Artist ID matching APIArtist.ID
	DatesLocations map[string][]string `json:"datesLocations"` // Raw concert location->dates mapping
}

// APIRelation wraps the complete concert relations dataset from the /api/relation endpoint.
//
// The API returns relations in an "index" wrapper array. This structure mirrors
// that exact format and is used only during initial data loading.
type APIRelation struct {
	Index []APIRelationIndex `json:"index"` // Array of all artist concert relations
}

// --- Core Domain Models ---
//
// These structures represent the internal business logic of the application.
// They are enriched versions of the API data with computed fields, indexing,
// and application-specific functionality for filtering, navigation, and display.

// Artist represents the complete internal model of a music artist/band.
//
// This is the rich domain model used throughout the application, containing both
// original API data and computed fields for enhanced functionality:
//
// Performance: Pre-computed fields like ConcertCount avoid repeated calculations
// Filtering: Countries slice enables efficient country-based filtering
// SEO: Slug field provides URL-friendly artist identification
// Caching: DatesAtLocation map enables fast location-specific concert lookups
// Navigation: On-demand adjacent artist lookup via GetAdjacentArtists()
type Artist struct {
	ID              int                 // Unique identifier matching API data
	Name            string              // Artist/band name for display
	Slug            string              // URL-friendly identifier (e.g., "queen", "led-zeppelin")
	Members         []string            // Current band member names
	CreationYear    int                 // Band formation year
	FirstAlbum      string              // First album date string (various formats)
	Image           string              // Artist image URL for display
	Concerts        []Concert           // Structured concert events (processed from API relations)
	DatesAtLocation map[string][]string // Pre-indexed concert dates by location slug for fast lookups
	ConcertCount    int                 // Total number of concerts (computed field)
	Countries       []string            // Unique countries where artist performed (sorted, for filtering)
}

// ArtistAtLocation represents an artist's concert activity at a specific venue.
//
// This structure is used in Location objects to track which artists have performed
// at each venue and how many times. Enables efficient queries like "top artists at venue"
// and supports location detail pages with artist statistics.
type ArtistAtLocation struct {
	Artist       Artist // Full artist information for display
	ConcertCount int    // Number of concerts this artist held at the location
}

// Location represents the complete internal model of a concert venue.
//
// Like Artist, this is a rich domain model with both source data and computed fields:
//
// Performance: Pre-computed aggregates like TotalConcerts and ArtistCount
// Temporal: EarliestYear/LatestYear enable decade-based filtering
// SEO: Slug field provides URL-friendly location identification
// Analysis: Artists slice enables detailed venue analytics and rankings
type Location struct {
	Name          string             // Human-readable location name (e.g., "London UK")
	Slug          string             // URL-friendly identifier (e.g., "london-uk")
	Artists       []ArtistAtLocation // Artists who performed here with concert counts
	ArtistCount   int                // Number of unique artists (computed field)
	TotalConcerts int                // Total concerts held here (computed field)
	EarliestYear  int                // Year of first concert at this location
	LatestYear    int                // Year of most recent concert at this location
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
// This structure is computed by analyzing the current artist dataset to determine:
//   - Realistic bounds for range sliders (no impossible values)
//   - Available discrete options for checkboxes (only values present in data)
//
// Frontend components use this data to configure filter UI elements dynamically,
// ensuring users can't set filter combinations that would return empty results.
type ArtistFilterOptions struct {
	// Range bounds for dual-range slider components
	CreationYearMin   int `json:"creationYearMin"`   // Earliest band formation year in dataset
	CreationYearMax   int `json:"creationYearMax"`   // Latest band formation year in dataset
	FirstAlbumYearMin int `json:"firstAlbumYearMin"` // Earliest first album year in dataset
	FirstAlbumYearMax int `json:"firstAlbumYearMax"` // Latest first album year in dataset

	// Available options for checkbox components
	MemberCounts []int    `json:"memberCounts"` // All member counts found in dataset (sorted)
	Countries    []string `json:"countries"`    // All countries from concert locations (sorted)
}

// LocationFilterParams represents all possible filter criteria for location searches.
//
// Similar structure to ArtistFilterParams but with location-specific criteria:
//   - Concert volume: How many concerts were held at the location
//   - Artist diversity: How many different artists performed there
//   - Temporal range: When concerts occurred at the location
//   - Geographic: Country-based location filtering
//
// Enables complex location queries like "venues with 10+ concerts by 5+ artists in the 1980s".
type LocationFilterParams struct {
	// Range filters for location characteristics
	ConcertCountFrom *int `json:"concertCountFrom,omitempty"` // Minimum total concerts held (inclusive)
	ConcertCountTo   *int `json:"concertCountTo,omitempty"`   // Maximum total concerts held (inclusive)

	ArtistCountFrom *int `json:"artistCountFrom,omitempty"` // Minimum unique artists performed (inclusive)
	ArtistCountTo   *int `json:"artistCountTo,omitempty"`   // Maximum unique artists performed (inclusive)

	ConcertYearFrom *int `json:"concertYearFrom,omitempty"` // Earliest concert year (inclusive)
	ConcertYearTo   *int `json:"concertYearTo,omitempty"`   // Latest concert year (inclusive)

	// Geographic filtering
	Countries []string `json:"countries,omitempty"` // Countries where location must be situated
}

// LocationFilterOptions represents the complete set of available filter options for locations.
//
// Computed by analyzing current location dataset to provide realistic filter bounds
// and available geographic options. Used to configure location filtering UI components.
type LocationFilterOptions struct {
	// Range bounds for slider components
	ConcertCountMin int `json:"concertCountMin"` // Minimum concert count across all locations
	ConcertCountMax int `json:"concertCountMax"` // Maximum concert count across all locations
	ArtistCountMin  int `json:"artistCountMin"`  // Minimum artist count across all locations
	ArtistCountMax  int `json:"artistCountMax"`  // Maximum artist count across all locations
	ConcertYearMin  int `json:"concertYearMin"`  // Earliest concert year across all locations
	ConcertYearMax  int `json:"concertYearMax"`  // Latest concert year across all locations

	// Available options for checkbox components
	Countries []string `json:"countries"` // All countries from location names (sorted)
}

// --- Search Data Structures ---
//
// These models define the search functionality that allows users to search across
// all data types with suggestions and type identification. The search system
// provides case-insensitive matching across artists, members, locations, and dates.

// SearchSuggestionType represents the type of search result for categorization.
type SearchSuggestionType string

const (
	SuggestionTypeArtist     SearchSuggestionType = "artist"      // Artist/band name
	SuggestionTypeMember     SearchSuggestionType = "member"      // Band member name
	SuggestionTypeLocation   SearchSuggestionType = "location"    // Concert location
	SuggestionTypeFirstAlbum SearchSuggestionType = "first-album" // First album date
	SuggestionTypeCreation   SearchSuggestionType = "creation"    // Band creation date
)

// SearchSuggestion represents a single search suggestion with type identification.
//
// This structure provides the frontend with enough information to display
// meaningful suggestions with context about what type of data was matched.
// The URL field enables direct navigation to relevant detail pages.
type SearchSuggestion struct {
	Text        string               `json:"text"`        // Display text for the suggestion
	Type        SearchSuggestionType `json:"type"`        // Type of match for categorization
	Description string               `json:"description"` // Additional context (e.g., "Queen - artist")
	URL         string               `json:"url"`         // Direct link to detail page
	ArtistID    int                  `json:"artistId"`    // Related artist ID for context
}

// SearchResult represents a comprehensive search result with matched items.
//
// Contains all matching artists and the search parameters used. This structure
// supports both direct search results and integration with existing filter system.
type SearchResult struct {
	Artists      []Artist `json:"artists"`      // Artists matching search criteria
	Query        string   `json:"query"`        // Original search query
	TotalResults int      `json:"totalResults"` // Number of matching results
}

// SearchParams represents search query parameters from user input.
//
// Supports integration with existing filter system by combining text search
// with advanced filtering options for more precise results.
type SearchParams struct {
	Query   string             `json:"query"`   // Search text input
	Filters ArtistFilterParams `json:"filters"` // Optional additional filters
}
