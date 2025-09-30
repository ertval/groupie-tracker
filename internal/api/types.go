package api

// Artist represents the raw artist payload returned by the Groupie Tracker API.
type Artist struct {
	ID           int      `json:"id"`
	Name         string   `json:"name"`
	Members      []string `json:"members"`
	CreationYear int      `json:"creationDate"`
	FirstAlbum   string   `json:"firstAlbum"`
	Image        string   `json:"image"`
}

// RelationIndex contains the concert schedule for a single artist from the /relation endpoint.
type RelationIndex struct {
	ID             int                 `json:"id"`
	DatesLocations map[string][]string `json:"datesLocations"`
}

// Relation is the container returned by the /relation endpoint.
type Relation struct {
	Index []RelationIndex `json:"index"`
}
