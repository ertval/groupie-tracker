package data

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// SearchIndex holds normalized search tokens for fast searching.
// Built during Catalog.Build() and immutable afterward.
type SearchIndex struct {
	artistTokens   map[int][]string    // artistID -> normalized tokens
	locationTokens map[string][]string // locationSlug -> normalized tokens
}

// Catalog is a lightweight component that owns and provides access to normalized data.
// It maintains the core collections (Artists, Locations, Concerts) and provides
// simple query methods for accessing them. After Build() is called, all data is
// immutable and safe for concurrent reads without locking.
type Catalog struct {
	// Core collections - immutable after Build()
	Artists   []*Artist           // All artists sorted alphabetically by name
	Locations map[string]Location // Locations indexed by slug for O(1) lookup
	Concerts  []Concert           // All concerts (may be redundant with Artist.Concerts)

	// Lookup indexes for fast access
	artistsByID     map[int]*Artist     // O(1) lookup by artist ID
	artistsBySlug   map[string]*Artist  // O(1) lookup by URL-friendly slug
	artistPositions map[int]int         // Maps artist ID to its index in sorted Artists slice
	locationsBySlug map[string]Location // Same as Locations (for backward compatibility)

	// Search index - immutable after Build()
	searchIndex *SearchIndex // Token-based search index for fast queries
}

// NewCatalog creates a new empty Catalog ready for building.
func NewCatalog() *Catalog {
	return &Catalog{
		Artists:         make([]*Artist, 0),
		Locations:       make(map[string]Location),
		Concerts:        make([]Concert, 0),
		artistsByID:     make(map[int]*Artist),
		artistsBySlug:   make(map[string]*Artist),
		artistPositions: make(map[int]int),
		locationsBySlug: make(map[string]Location),
	}
}

// AddArtist adds an artist to the catalog. Should be called before Build().
func (c *Catalog) AddArtist(artist *Artist) {
	c.Artists = append(c.Artists, artist)
	// Concerts will be extracted during Build()
}

// AddConcert adds a concert to the catalog and associates it with the artist.
// Should be called before Build().
func (c *Catalog) AddConcert(concert Concert) {
	c.Concerts = append(c.Concerts, concert)

	// Also add to the artist's concert list
	if artist := c.findArtistByID(concert.ArtistID); artist != nil {
		artist.Concerts = append(artist.Concerts, concert)
	}
}

// findArtistByID is a helper for pre-Build lookups (linear search).
func (c *Catalog) findArtistByID(id int) *Artist {
	for _, artist := range c.Artists {
		if artist.ID == id {
			return artist
		}
	}
	return nil
}

// Build finalizes the catalog by sorting, building indexes, and aggregating locations.
// After this call, the catalog is immutable and safe for concurrent reads.
func (c *Catalog) Build() error {
	// Sort artists alphabetically by name
	sort.Slice(c.Artists, func(i, j int) bool {
		return c.Artists[i].Name < c.Artists[j].Name
	})

	// Build artist indexes
	for i, artist := range c.Artists {
		c.artistsByID[artist.ID] = artist
		c.artistsBySlug[artist.Slug()] = artist
		c.artistPositions[artist.ID] = i

		// Extract concerts from artists into catalog.Concerts for location aggregation
		c.Concerts = append(c.Concerts, artist.Concerts...)
	}

	// Build location aggregations
	if err := c.buildLocations(); err != nil {
		return err
	}

	// Build search index
	c.searchIndex = c.buildSearchIndex()

	return nil
}

// buildLocations aggregates concerts by location and builds Location structs.
func (c *Catalog) buildLocations() error {
	// Group concerts by location slug
	concertsByLocation := make(map[string][]Concert)
	for _, concert := range c.Concerts {
		slug := concert.LocationSlug
		concertsByLocation[slug] = append(concertsByLocation[slug], concert)
	}

	// Create Location objects
	for slug, concerts := range concertsByLocation {
		if len(concerts) == 0 {
			continue
		}

		// Get the original location name from the first concert
		locationName := concerts[0].Location

		// Group concerts by artist for ArtistAtLocation
		artistConcerts := make(map[int][]Concert)
		for _, concert := range concerts {
			artistConcerts[concert.ArtistID] = append(artistConcerts[concert.ArtistID], concert)
		}

		// Build ArtistAtLocation slice
		artistsAtLocation := make([]ArtistAtLocation, 0, len(artistConcerts))
		for artistID, concertsForArtist := range artistConcerts {
			artist := c.artistsByID[artistID]
			if artist != nil {
				artistsAtLocation = append(artistsAtLocation, ArtistAtLocation{
					Artist:       artist,
					ConcertCount: len(concertsForArtist),
				})
			}
		}

		// Sort artists by name for consistent ordering
		sort.Slice(artistsAtLocation, func(i, j int) bool {
			return artistsAtLocation[i].Artist.Name < artistsAtLocation[j].Artist.Name
		})

		location := Location{
			Name:    locationName,
			Slug:    slug,
			Artists: artistsAtLocation,
		}

		c.Locations[slug] = location
		c.locationsBySlug[slug] = location
	}

	return nil
}

// ArtistByID retrieves an artist by ID. Returns error if not found.
func (c *Catalog) ArtistByID(id int) (*Artist, error) {
	artist, ok := c.artistsByID[id]
	if !ok {
		return nil, fmt.Errorf("artist with ID %d not found", id)
	}
	return artist, nil
}

// ArtistBySlug retrieves an artist by slug. Returns error if not found.
func (c *Catalog) ArtistBySlug(slug string) (*Artist, error) {
	artist, ok := c.artistsBySlug[slug]
	if !ok {
		return nil, fmt.Errorf("artist with slug %q not found", slug)
	}
	return artist, nil
}

// LocationBySlug retrieves a location by slug. Returns error if not found.
func (c *Catalog) LocationBySlug(slug string) (Location, error) {
	location, ok := c.locationsBySlug[slug]
	if !ok {
		return Location{}, fmt.Errorf("location with slug %q not found", slug)
	}
	return location, nil
}

// AllArtists returns all artists sorted alphabetically.
func (c *Catalog) AllArtists() []*Artist {
	return c.Artists
}

// AllLocations returns all locations sorted by total concert count (descending).
func (c *Catalog) AllLocations() []Location {
	locations := make([]Location, 0, len(c.Locations))
	for _, location := range c.Locations {
		locations = append(locations, location)
	}

	// Sort by total concerts descending, then by name for ties
	sort.Slice(locations, func(i, j int) bool {
		countI := locations[i].TotalConcerts()
		countJ := locations[j].TotalConcerts()
		if countI != countJ {
			return countI > countJ // Descending by concert count
		}
		return locations[i].Name < locations[j].Name // Ascending by name for ties
	})

	return locations
}

// ArtistPosition returns the position of an artist in the sorted Artists slice.
// Returns -1 if the artist is not found. This is useful for adjacent navigation.
func (c *Catalog) ArtistPosition(id int) int {
	if pos, ok := c.artistPositions[id]; ok {
		return pos
	}
	return -1
}

// normalizeTokens converts text to normalized search tokens.
// Tokens are lowercase, with special characters removed, deduplicated.
func normalizeTokens(text string) []string {
	// Convert to lowercase
	text = strings.ToLower(strings.TrimSpace(text))

	// Remove special characters, keep only alphanumeric and spaces
	reg := regexp.MustCompile(`[^a-z0-9\s]+`)
	text = reg.ReplaceAllString(text, " ")

	// Split by whitespace and deduplicate
	words := strings.Fields(text)
	seen := make(map[string]bool)
	tokens := make([]string, 0, len(words))

	for _, word := range words {
		if word != "" && !seen[word] {
			tokens = append(tokens, word)
			seen[word] = true
		}
	}

	return tokens
}

// buildSearchIndex creates the search index with normalized tokens for all searchable entities.
func (c *Catalog) buildSearchIndex() *SearchIndex {
	index := &SearchIndex{
		artistTokens:   make(map[int][]string, len(c.Artists)),
		locationTokens: make(map[string][]string, len(c.Locations)),
	}

	// Index artists
	for _, artist := range c.Artists {
		tokens := make([]string, 0)

		// Add artist name tokens
		tokens = append(tokens, normalizeTokens(artist.Name)...)

		// Add member name tokens
		for _, member := range artist.Members {
			tokens = append(tokens, normalizeTokens(member)...)
		}

		// Add creation year as token
		tokens = append(tokens, strconv.Itoa(artist.CreationYear))

		// Add first album tokens
		tokens = append(tokens, normalizeTokens(artist.FirstAlbum)...)

		// Add location tokens (countries and city names)
		for _, country := range artist.Countries() {
			tokens = append(tokens, normalizeTokens(country)...)
		}

		for _, concert := range artist.Concerts {
			tokens = append(tokens, normalizeTokens(concert.Location)...)
		}

		// Deduplicate tokens
		seen := make(map[string]bool)
		uniqueTokens := make([]string, 0, len(tokens))
		for _, token := range tokens {
			if token != "" && !seen[token] {
				uniqueTokens = append(uniqueTokens, token)
				seen[token] = true
			}
		}

		index.artistTokens[artist.ID] = uniqueTokens
	}

	// Index locations
	for slug, location := range c.Locations {
		tokens := normalizeTokens(location.Name)
		index.locationTokens[slug] = tokens
	}

	return index
}
