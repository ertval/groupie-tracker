package view

import (
	"groupie-tracker/internal/data"
)

// Page represents the base structure for all page views.
// It contains common metadata that every page needs for rendering.
type Page struct {
	Title       string                  // Page title for <title> tag
	ExtraCSS    string                  // Additional CSS file to load (e.g., "home.css")
	ExtraJS     string                  // Additional JavaScript file to load
	Suggestions []data.SearchSuggestion // Search suggestions for autocomplete
	Error       *ErrorInfo              // Error information if page is an error page
}

// ErrorInfo represents error details for error pages.
type ErrorInfo struct {
	Code         int    // HTTP status code (e.g., 404, 500)
	Message      string // Error message to display
	RequestedURL string // URL that was requested
	Timestamp    string // Time when error occurred
}

// HomePage represents the home page view.
type HomePage struct {
	Page
	Artists        []*data.Artist // Featured/random artists to display
	TotalMembers   int            // Total members across all artists
	TotalLocations int            // Total unique locations
}

// ArtistListPage represents the artists listing page view.
type ArtistListPage struct {
	Page
	Artists        []*data.Artist           // List of artists to display
	FilterOptions  data.ArtistFilterOptions // Available filter values
	AppliedFilters data.ArtistFilterParams  // Currently applied filters
	IsFiltered     bool                     // Whether filters are active
	TotalArtists   int                      // Total artists before filtering
}

// ArtistDetailPage represents an individual artist detail page view.
type ArtistDetailPage struct {
	Page
	Artist     *data.Artist // The artist being displayed
	PrevArtist *data.Artist // Previous artist for navigation
	NextArtist *data.Artist // Next artist for navigation
}

// LocationListPage represents the locations listing page view.
type LocationListPage struct {
	Page
	Locations         []data.Location            // List of locations to display
	FilterOptions     data.LocationFilterOptions // Available filter values
	AppliedFilters    data.LocationFilterParams  // Currently applied filters
	IsFiltered        bool                       // Whether filters are active
	FilterDescription string                     // Human-readable filter description
	TotalLocations    int                        // Total locations before filtering
	TotalCountries    int                        // Total unique countries
	TotalConcerts     int                        // Total concerts across all locations
}

// LocationDetailPage represents an individual location detail page view.
type LocationDetailPage struct {
	Page
	Location     data.Location           // The location being displayed
	Artists      []data.ArtistAtLocation // Artists who performed at this location
	PrevLocation *data.Location          // Previous location for navigation
	NextLocation *data.Location          // Next location for navigation
}

// SearchPage represents the search page view.
type SearchPage struct {
	Page
	Query          string                   // Search query entered by user
	Results        data.SearchResult        // Search results
	FilterOptions  data.ArtistFilterOptions // Available filter values
	AppliedFilters data.ArtistFilterParams  // Currently applied filters
	IsSearch       bool                     // Whether a search was performed
}

// DevPage represents the developer tools page view.
type DevPage struct {
	Page
	Links []DevLink // List of developer tool links
}

// DevLink represents a link on the developer page.
type DevLink struct {
	Href string // URL of the link
	Text string // Display text for the link
}

// HealthResponse represents the health check API response.
type HealthResponse struct {
	Status    string        `json:"status"`    // Health status (e.g., "healthy")
	Timestamp string        `json:"timestamp"` // Current timestamp
	Stats     data.AppStats `json:"stats"`     // Application statistics
}
