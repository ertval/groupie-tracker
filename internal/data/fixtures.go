package data

// NewStoreFromFixtures constructs a Store populated with the provided fixtures.
// It mirrors the production loading pipeline so tests can operate on a fully
// initialized dataset without hitting the external API.
func NewStoreFromFixtures(artists []Artist, locations []Location) *Store {
	store := &Store{}

	normalizedArtists := make([]*Artist, len(artists))
	for i := range artists {
		artist := artists[i]
		normalizedArtists[i] = &artist
	}

	store.artists = normalizedArtists
	store.artistsByID, store.artistsBySlug, store.artistPositions = store.createArtistIndexes(store.artists)

	if len(locations) > 0 {
		store.locations = make([]Location, len(locations))
		copy(store.locations, locations)

		store.locationsBySlug = make(map[string]Location, len(store.locations))
		for i := range store.locations {
			if store.locations[i].Slug == "" {
				store.locations[i].Slug = createSlug(store.locations[i].Name)
			}
			store.locationsBySlug[store.locations[i].Slug] = store.locations[i]
		}
	} else {
		store.locations, store.locationsBySlug = store.createLocationsData(store.artists)
	}

	store.artistFilters = store.calculateArtistFilterOptions(store.artists)
	store.locationFilters = store.calculateLocationFilterOptions(store.locations)
	store.suggestions = store.generateSearchSuggestions(store.artists)
	store.appStats = store.calculateStats(store.artists, store.locations, 0, 0)
	store.cacheEnabled = false
	store.searchCache = make(map[string][]*Artist, 50)

	return store
}
