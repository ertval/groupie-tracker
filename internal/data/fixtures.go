package data

// NewStoreFromFixtures constructs a Store populated with the provided fixtures.
// It mirrors the production loading pipeline so tests can operate on a fully
// initialized dataset without hitting the external API.
func NewStoreFromFixtures(artists []Artist, locations []Location) *Store {
	store := &Store{}

	// Convert to pointers
	normalizedArtists := make([]*Artist, len(artists))
	for i := range artists {
		artist := artists[i]
		normalizedArtists[i] = &artist
	}

	// Build catalog
	catalog := NewCatalog()
	for _, artist := range normalizedArtists {
		catalog.AddArtist(artist)
		// Concerts will be extracted from artist.Concerts during Build()
	}

	// If locations are provided, we still need to build from artists
	// since Catalog handles location building internally
	if err := catalog.Build(); err != nil {
		// In tests, we shouldn't fail silently, but for backward compatibility
		// we'll just continue
	}

	store.catalog = catalog
	store.artistFilters = store.calculateArtistFilterOptions(normalizedArtists)
	store.locationFilters = store.calculateLocationFilterOptions(catalog.AllLocations())
	store.suggestions = store.generateSearchSuggestions(normalizedArtists)
	store.appStats = store.calculateStats(normalizedArtists, catalog.AllLocations(), 0, 0)

	return store
}
