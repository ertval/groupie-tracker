package api

// Raw API response models - these structs match the external API JSON structure exactly.
// No computed fields, transformations, or business logic - just pure data transfer objects.

// Artist represents a music artist as returned from the /api/artists endpoint.
// Field names use JSON tags to map API's snake_case/camelCase to Go's idiomatic naming.
type Artist struct {
	ID           int      `json:"id"`           // Unique identifier for the artist
	Name         string   `json:"name"`         // Artist or band name (e.g., "Queen", "Pink Floyd")
	Members      []string `json:"members"`      // List of member names (band members or solo artist)
	CreationYear int      `json:"creationDate"` // Year the band was formed (API uses "creationDate" key)
	FirstAlbum   string   `json:"firstAlbum"`   // First album release date in format "DD-MM-YYYY"
	Image        string   `json:"image"`        // URL to artist's official image (external CDN link)
}

// RelationIndex maps concert dates to locations for a single artist.
// The DatesLocations map uses location strings as keys (e.g., "london-uk") and date arrays as values.
type RelationIndex struct {
	ID             int                 `json:"id"`             // Artist ID matching the Artist.ID field
	DatesLocations map[string][]string `json:"datesLocations"` // Map of "location-country" to array of concert dates
}

// Relation wraps the complete concert relations dataset from the /api/relation endpoint.
// Contains an array of RelationIndex entries, one per artist with concert data.
type Relation struct {
	Index []RelationIndex `json:"index"` // Array of all artists' concert date-location mappings
}
