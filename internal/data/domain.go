
package data

// Artist is the rich internal representation of an artist.
type Artist struct {
	ID              int
	Name            string
	Slug            string
	Members         []string
	CreationYear    int
	FirstAlbum      string
	Image           string
	Concerts        []Concert // A slice of structured Concert objects
	DatesAtLocation map[string][]string // Pre-computed concert dates per location slug
	ConcertCount    int
	Countries       []string // Pre-sorted list of unique countries
	NextArtistID    int      // ID of the next artist (for navigation)
	PrevArtistID    int      // ID of the previous artist
}

// Location is the rich internal representation of a concert location.
type Location struct {
	Name          string
	Slug          string
	Artists       []Artist // Artists who have played here
	ArtistCount   int
	TotalConcerts int
}

// Concert represents a single concert event.
type Concert struct {
	Date     string
	Location string // Normalized location name
}
