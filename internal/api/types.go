package api

// APIArtist represents the raw artist data structure from the /api/artists endpoint.
// This is a direct mapping of the external API response with minimal processing.
type APIArtist struct {
	ID           int      `json:"id"`
	Name         string   `json:"name"`
	Members      []string `json:"members"`
	CreationDate int      `json:"creationDate"`
	FirstAlbum   string   `json:"firstAlbum"`
	Image        string   `json:"image"`
}

// APIRelationIndex represents a single artist's concert data from the /api/relation endpoint.
type APIRelationIndex struct {
	ID             int                 `json:"id"`
	DatesLocations map[string][]string `json:"datesLocations"`
}

// APIRelation wraps the complete concert relations dataset from the /api/relation endpoint.
type APIRelation struct {
	Index []APIRelationIndex `json:"index"`
}
