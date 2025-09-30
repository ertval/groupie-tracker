package data

// Concert represents a single concert occurrence for an artist.
type Concert struct {
	Location string `json:"location"`
	Country  string `json:"country"`
	Date     string `json:"date"`
	Year     int    `json:"year"`
}

// Artist is the internal domain model used throughout the application.
type Artist struct {
	ID           int       `json:"id"`
	Name         string    `json:"name"`
	Slug         string    `json:"slug"`
	Members      []string  `json:"members"`
	CreationYear int       `json:"creation_year"`
	FirstAlbum   string    `json:"first_album"`
	Image        string    `json:"image"`
	Concerts     []Concert `json:"concerts"`
	Countries    []string  `json:"countries"`
	ConcertCount int       `json:"concert_count"`
}

// Location aggregates concert metrics by venue/location.
type Location struct {
	Name          string    `json:"name"`
	Slug          string    `json:"slug"`
	Artists       []string  `json:"artists"`
	ArtistCount   int       `json:"artist_count"`
	TotalConcerts int       `json:"total_concerts"`
	EarliestYear  int       `json:"earliest_year"`
	LatestYear    int       `json:"latest_year"`
	Concerts      []Concert `json:"concerts"`
}

// AppStats contains derived statistics exposed on the home screen and health endpoint.
type AppStats struct {
	TotalArtists   int `json:"total_artists"`
	TotalLocations int `json:"total_locations"`
	TotalConcerts  int `json:"total_concerts"`
	EarliestYear   int `json:"earliest_year"`
	LatestYear     int `json:"latest_year"`
}
