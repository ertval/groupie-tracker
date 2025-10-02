package data

import (
	"sort"
	"time"
)

/* BACKUP - Original Artist struct before Phase 1 refactoring:
type Artist struct {
	ID              int
	Name            string
	Slug            string
	Members         []string
	CreationYear    int
	FirstAlbum      string
	Image           string
	Concerts        []Concert
	DatesAtLocation map[string][]string
	ConcertCount    int
	Countries       []string
	MemberCount     int
	FirstAlbumYear  int
}
*/

// Artist represents the complete internal model of a music artist.
// Removed cached/computed fields - use helper methods instead.
type Artist struct {
	ID           int
	Name         string
	Members      []string
	CreationYear int
	FirstAlbum   string
	Image        string
	Concerts     []Concert // New: holds all concert events for this artist
}

// MemberCount returns the number of members in the band.
func (a *Artist) MemberCount() int {
	return len(a.Members)
}

// ConcertCount returns the total number of concerts for this artist.
func (a *Artist) ConcertCount() int {
	return len(a.Concerts)
}

// FirstAlbumYear extracts and returns the year from the FirstAlbum date string.
// Returns 0 if the year cannot be parsed.
func (a *Artist) FirstAlbumYear() int {
	return extractYearFromDate(a.FirstAlbum)
}

// Countries returns a sorted list of unique countries where the artist has performed.
func (a *Artist) Countries() []string {
	if len(a.Concerts) == 0 {
		return []string{}
	}

	countryMap := make(map[string]bool)
	for _, concert := range a.Concerts {
		country := extractCountryFromLocation(concert.Location)
		if country != "" {
			countryMap[country] = true
		}
	}

	countries := make([]string, 0, len(countryMap))
	for country := range countryMap {
		countries = append(countries, country)
	}
	sort.Strings(countries)
	return countries
}

// Slug returns a URL-friendly identifier for the artist.
func (a *Artist) Slug() string {
	return createSlug(a.Name)
}

// DatesAtLocation returns concert dates grouped by location slug.
// This is computed on-demand for backward compatibility.
func (a *Artist) DatesAtLocation() map[string][]string {
	result := make(map[string][]string)
	for _, concert := range a.Concerts {
		slug := createSlug(concert.Location)
		result[slug] = append(result[slug], concert.DateString)
	}
	return result
}

// ArtistAtLocation represents an artist's concert activity at a specific venue.
type ArtistAtLocation struct {
	Artist       *Artist
	ConcertCount int
}

// Location represents the complete internal model of a concert venue.
type Location struct {
	Name    string
	Slug    string
	Artists []ArtistAtLocation
	// Years can be computed from concerts
}

// Country extracts and returns the country from the location name.
func (l Location) Country() string {
	return extractCountryFromLocation(l.Name)
}

// ArtistCount returns the number of unique artists that performed at this location.
func (l Location) ArtistCount() int {
	return len(l.Artists)
}

// TotalConcerts returns the total number of concerts at this location.
func (l Location) TotalConcerts() int {
	total := 0
	for _, artist := range l.Artists {
		total += artist.ConcertCount
	}
	return total
}

// YearRange returns the earliest and latest concert years at this location.
func (l Location) YearRange() (int, int) {
	if len(l.Artists) == 0 {
		return 0, 0
	}

	earliest := 9999
	latest := 0

	// Get years from all concerts at this location
	for _, artistAtLoc := range l.Artists {
		for _, concert := range artistAtLoc.Artist.Concerts {
			// Match by location slug since Concert stores normalized location
			if createSlug(concert.Location) != l.Slug {
				continue
			}
			if !concert.Date.IsZero() {
				year := concert.Date.Year()
				if year < earliest {
					earliest = year
				}
				if year > latest {
					latest = year
				}
			}
		}
	}

	if earliest == 9999 {
		return 0, 0
	}
	return earliest, latest
} // Concert represents a single concert event in structured form.
// This simplified structure is created by parsing the complex API relations data.
// Each Concert represents one performance at one venue on one date, making it
// easy to filter, count, and analyze concert data across the application.
type Concert struct {
	ArtistID     int       // Artist performing at this concert
	Location     string    // Normalized location name
	LocationSlug string    // URL-friendly location slug
	Date         time.Time // Parsed concert date
	DateString   string    // Original date string for display
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
	CreationYear IntRange `json:"creationYear"`
	FirstAlbum   IntRange `json:"firstAlbum"`
	MemberCounts []int    `json:"memberCounts"`
	Countries    []string `json:"countries"`
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
	ConcertCount IntRange `json:"concertCount"`
	ArtistCount  IntRange `json:"artistCount"`
	YearRange    IntRange `json:"yearRange"`
	Countries    []string `json:"countries"`
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
	Artists      []*Artist `json:"artists"`
	Query        string    `json:"query"`
	TotalResults int       `json:"totalResults"`
}

// SearchParams represents search query parameters from user input.
//
// Supports integration with existing filter system by combining text search
// with advanced filtering options for more precise results.
type SearchParams struct {
	Query   string             `json:"query"`   // Search text input
	Filters ArtistFilterParams `json:"filters"` // Optional additional filters
}

// storeStats represents application-wide statistics with type-safe fields.
//
// This structure provides a type-safe alternative to the map[string]int approach,
// enabling compile-time validation and better API documentation. All fields
// represent counts computed during data loading or runtime.
type storeStats struct {
	TotalArtists     int `json:"total_artists"`     // Number of artists in the dataset
	TotalMembers     int `json:"total_members"`     // Sum of all band members across all artists
	TotalLocations   int `json:"total_locations"`   // Number of unique concert venues
	TotalConcerts    int `json:"total_concerts"`    // Total number of concert events
	TotalCountries   int `json:"total_countries"`   // Number of unique countries with concerts
	CachedImages     int `json:"cached_images"`     // Number of artist images served from local cache
	DownloadedImages int `json:"downloaded_images"` // Number of artist images downloaded this session
}

// AppStats is a public type alias for storeStats
type AppStats = storeStats
