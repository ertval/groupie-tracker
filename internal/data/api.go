
package data

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
