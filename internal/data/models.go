package data

// == API models representing the JSON structure returned by the external API. ==

// APIArtist represents a single artist record from the /api/artists endpoint.
type APIArtist struct {
	ID           int      `json:"id"`
	Name         string   `json:"name"`
	Members      []string `json:"members"`
	CreationYear int      `json:"creationDate"`
	FirstAlbum   string   `json:"firstAlbum"`
	Image        string   `json:"image"`
}

// APIRelationIndex represents a single entry in the relation index.
type APIRelationIndex struct {
	ID             int                 `json:"id"`
	DatesLocations map[string][]string `json:"datesLocations"`
}

// APIRelation represents the concert relations for all artists from the /api/relation endpoint.
type APIRelation struct {
	Index []APIRelationIndex `json:"index"`
}

// == Domain models and data structures used internally by the application. ==

// Artist is the rich internal representation of an artist.
type Artist struct {
	ID              int
	Name            string
	Slug            string
	Members         []string
	CreationYear    int
	FirstAlbum      string
	Image           string
	Concerts        []Concert           // A slice of structured Concert objects
	DatesAtLocation map[string][]string // Pre-computed concert dates per location slug
	ConcertCount    int
	Countries       []string // Pre-sorted list of unique countries
	NextArtistID    int      // ID of the next artist (for navigation)
	PrevArtistID    int      // ID of the previous artist
}

// ArtistAtLocation represents an artist and their concert count at a specific location.
type ArtistAtLocation struct {
	Artist       Artist
	ConcertCount int
}

// Location is the rich internal representation of a concert location.
type Location struct {
	Name          string
	Slug          string
	Artists       []ArtistAtLocation // Artists who have played here with their concert counts
	ArtistCount   int
	TotalConcerts int
}

// Concert represents a single concert event.
type Concert struct {
	Date     string
	Location string // Normalized location name
}

// CacheStatus represents the state of the image cache.
type CacheStatus int

const (
	// CacheDisabled means image caching is turned off.
	CacheDisabled CacheStatus = iota
	// CacheCold means caching is on, but the cache was empty and images were downloaded.
	CacheCold
	// CacheWarm means caching is on, and images were served from the existing cache.
	CacheWarm
)

// == Filter-related data structures ==

// FilterParams represents the filter criteria for artist searches
type FilterParams struct {
	// Range filters (using sliders)
	CreationYearFrom *int `json:"creationYearFrom,omitempty"`
	CreationYearTo   *int `json:"creationYearTo,omitempty"`

	FirstAlbumYearFrom *int `json:"firstAlbumYearFrom,omitempty"`
	FirstAlbumYearTo   *int `json:"firstAlbumYearTo,omitempty"`

	// Checkbox filters
	MemberCounts []int    `json:"memberCounts,omitempty"` // Changed to checkbox list
	Countries    []string `json:"countries,omitempty"`    // Changed to countries instead of cities
}

// FilterOptions represents the available filter options
type FilterOptions struct {
	// Range bounds for sliders
	CreationYearMin   int `json:"creationYearMin"`
	CreationYearMax   int `json:"creationYearMax"`
	FirstAlbumYearMin int `json:"firstAlbumYearMin"`
	FirstAlbumYearMax int `json:"firstAlbumYearMax"`

	// Available checkbox options
	MemberCounts []int    `json:"memberCounts"` // List of available member counts (1 to max)
	Countries    []string `json:"countries"`    // List of countries extracted from locations
}

// LocationFilterParams represents the filter criteria for location searches
type LocationFilterParams struct {
	// Range filters
	ConcertCountFrom *int `json:"concertCountFrom,omitempty"`
	ConcertCountTo   *int `json:"concertCountTo,omitempty"`

	ArtistCountFrom *int `json:"artistCountFrom,omitempty"`
	ArtistCountTo   *int `json:"artistCountTo,omitempty"`

	// Checkbox filters
	Countries []string `json:"countries,omitempty"`
}

// LocationFilterOptions represents the available location filter options
type LocationFilterOptions struct {
	// Range bounds for sliders
	ConcertCountMin int `json:"concertCountMin"`
	ConcertCountMax int `json:"concertCountMax"`
	ArtistCountMin  int `json:"artistCountMin"`
	ArtistCountMax  int `json:"artistCountMax"`

	// Available checkbox options
	Countries []string `json:"countries"` // List of countries extracted from locations
}
