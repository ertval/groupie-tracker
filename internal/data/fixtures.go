package data

// NewStoreFromFixtures constructs a Store populated with the provided fixtures.
// It mirrors the production loading pipeline so tests can operate on a fully
// initialized dataset without hitting the external API.
func NewStoreFromFixtures(artists []Artist, locations []Location) *Store {
	store := &Store{}

	normalizedArtists := make([]Artist, len(artists))
	for i := range artists {
		artist := artists[i]
		if artist.Slug == "" {
			artist.Slug = createSlug(artist.Name)
		}

		if artist.MemberCount == 0 {
			artist.MemberCount = len(artist.Members)
		}

		if artist.FirstAlbumYear == 0 {
			artist.FirstAlbumYear = extractYearFromDate(artist.FirstAlbum)
		}

		if artist.DatesAtLocation == nil {
			artist.DatesAtLocation = make(map[string][]string)
		}

		countries := make(map[string]bool)
		for _, concert := range artist.Concerts {
			locationName := concert.Location
			if locationName == "" {
				continue
			}

			normalizedLocation := normalizeLocation(locationName)
			locationSlug := createSlug(normalizedLocation)
			artist.DatesAtLocation[locationSlug] = append(artist.DatesAtLocation[locationSlug], concert.Date)

			if country := extractCountryFromLocation(locationName); country != "" {
				countries[country] = true
			}
		}

		if len(artist.Countries) == 0 {
			artist.Countries = store.convertCountriesMapToSlice(countries)
		}
		artist.ConcertCount = len(artist.Concerts)

		normalizedArtists[i] = artist
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
			if store.locations[i].Country == "" {
				store.locations[i].Country = extractCountryFromLocation(store.locations[i].Name)
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

	return store
}
