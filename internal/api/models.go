package api

// Raw API response models - match API exactly, no computed fields

// Artist represents the raw artist data from the API.
type Artist struct {
	ID           int      `json:"id"`
	Name         string   `json:"name"`
	Members      []string `json:"members"`
	CreationYear int      `json:"creationDate"`
	FirstAlbum   string   `json:"firstAlbum"`
	Image        string   `json:"image"`
}

// RelationIndex represents the dates and locations for a single artist.
type RelationIndex struct {
	ID             int                 `json:"id"`
	DatesLocations map[string][]string `json:"datesLocations"`
}

// Relation represents the complete relation data from the API.
type Relation struct {
	Index []RelationIndex `json:"index"`
}
