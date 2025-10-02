package data

import "sort"

// FilterArtists applies user-specified filter criteria to the artist collection and returns matching artists.
// Filters support: creation year range, first album year range, member count values, and countries.
// All criteria are ANDed together (artist must match ALL specified filters to be included).
func (s *Store) FilterArtists(criteria ArtistFilterParams) []Artist {
	artists := s.Artists()
	if len(artists) == 0 {
		return nil
	}

	var filtered []Artist
	for _, artist := range artists {
		if matchesArtistFilters(artist, criteria) { // Check if artist matches all filter criteria
			filtered = append(filtered, artist)
		}
	}

	return filtered
}

// FilterLocations applies user-specified filter criteria to the location collection and returns matching locations.
// Filters support: concert count range, artist count range, year range (earliest/latest), and country.
// All criteria are ANDed together (location must match ALL specified filters to be included).
func (s *Store) FilterLocations(params LocationFilterParams) []Location {
	locations := s.Locations()
	if len(locations) == 0 {
		return nil
	}

	var filtered []Location
	for _, location := range locations {
		if matchesLocationFilters(location, params) { // Check if location matches all filter criteria
			filtered = append(filtered, location)
		}
	}

	return filtered
}

// matchesArtistFilters checks whether a single artist satisfies all specified filter criteria.
// Returns true only if the artist matches ALL non-nil/non-empty filter parameters (AND logic).
func matchesArtistFilters(artist Artist, params ArtistFilterParams) bool {
	// Filter by creation year range (e.g., "Show bands formed between 1970-1980")
	if params.CreationYearFrom != nil && artist.CreationYear < *params.CreationYearFrom {
		return false
	}
	if params.CreationYearTo != nil && artist.CreationYear > *params.CreationYearTo {
		return false
	}

	// Filter by first album year range (only check if artist has a valid album year > 0)
	if params.FirstAlbumYearFrom != nil || params.FirstAlbumYearTo != nil {
		albumYear := artist.FirstAlbumYear
		if albumYear > 0 { // Only apply filter if artist has a valid first album year
			if params.FirstAlbumYearFrom != nil && albumYear < *params.FirstAlbumYearFrom {
				return false
			}
			if params.FirstAlbumYearTo != nil && albumYear > *params.FirstAlbumYearTo {
				return false
			}
		}
	}

	// Filter by member count (e.g., "Show solo artists (1) or bands with 4 members")
	// This is an OR within the member counts list, but AND with other filters
	if len(params.MemberCounts) > 0 {
		memberCount := artist.MemberCount
		found := false
		for _, allowedCount := range params.MemberCounts {
			if memberCount == allowedCount {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// Filter by countries (e.g., "Show artists who performed in USA or UK")
	// Artist must have performed in at least ONE of the specified countries (OR within countries, AND with other filters)
	if len(params.Countries) > 0 {
		allowed := make(map[string]struct{}, len(params.Countries)) // Use map for O(1) lookup
		for _, country := range params.Countries {
			allowed[country] = struct{}{}
		}

		hasMatchingCountry := false
		for _, country := range artist.Countries {
			if _, ok := allowed[country]; ok {
				hasMatchingCountry = true
				break
			}
		}
		if !hasMatchingCountry {
			return false
		}
	}

	return true // Artist passed all filter checks
}

// matchesLocationFilters checks whether a single location satisfies all specified filter criteria.
// Returns true only if the location matches ALL non-nil/non-empty filter parameters (AND logic).
func matchesLocationFilters(location Location, params LocationFilterParams) bool {
	// Filter by concert count range (e.g., "Show locations with 10-50 concerts")
	if params.ConcertCountFrom != nil && location.TotalConcerts < *params.ConcertCountFrom {
		return false
	}
	if params.ConcertCountTo != nil && location.TotalConcerts > *params.ConcertCountTo {
		return false
	}

	if params.ArtistCountFrom != nil && location.ArtistCount < *params.ArtistCountFrom {
		return false
	}
	if params.ArtistCountTo != nil && location.ArtistCount > *params.ArtistCountTo {
		return false
	}

	if params.ConcertYearFrom != nil && location.LatestYear < *params.ConcertYearFrom {
		return false
	}
	if params.ConcertYearTo != nil && location.EarliestYear > *params.ConcertYearTo {
		return false
	}

	if len(params.Countries) > 0 {
		locationCountry := location.Country
		for _, allowedCountry := range params.Countries {
			if locationCountry == allowedCountry {
				return true
			}
		}
		return false
	}

	return true
}

// calculateArtistFilterOptions derives available artist filter metadata from the dataset.
func (s *Store) calculateArtistFilterOptions(artists []Artist) ArtistFilterOptions {
	if len(artists) == 0 {
		return ArtistFilterOptions{}
	}

	minCreationYear, maxCreationYear := artists[0].CreationYear, artists[0].CreationYear
	minFirstAlbumYear, maxFirstAlbumYear := 0, 0
	memberCountSet := make(map[int]bool)
	countrySet := make(map[string]bool)

	for _, artist := range artists {
		if artist.CreationYear < minCreationYear {
			minCreationYear = artist.CreationYear
		}
		if artist.CreationYear > maxCreationYear {
			maxCreationYear = artist.CreationYear
		}

		albumYear := artist.FirstAlbumYear
		if albumYear == 0 {
			albumYear = extractYearFromDate(artist.FirstAlbum)
		}
		if albumYear > 0 {
			if minFirstAlbumYear == 0 || albumYear < minFirstAlbumYear {
				minFirstAlbumYear = albumYear
			}
			if albumYear > maxFirstAlbumYear {
				maxFirstAlbumYear = albumYear
			}
		}

		memberCount := artist.MemberCount
		if memberCount == 0 {
			memberCount = len(artist.Members)
		}
		memberCountSet[memberCount] = true

		for _, country := range artist.Countries {
			if country != "" {
				countrySet[country] = true
			}
		}
	}

	memberCounts := make([]int, 0, len(memberCountSet))
	for count := range memberCountSet {
		memberCounts = append(memberCounts, count)
	}
	sort.Ints(memberCounts)

	countries := make([]string, 0, len(countrySet))
	for country := range countrySet {
		countries = append(countries, country)
	}
	sort.Strings(countries)

	if minFirstAlbumYear == 0 {
		minFirstAlbumYear = minCreationYear
	}
	if maxFirstAlbumYear == 0 {
		maxFirstAlbumYear = maxCreationYear
	}

	return ArtistFilterOptions{
		CreationYearMin:   minCreationYear,
		CreationYearMax:   maxCreationYear,
		FirstAlbumYearMin: minFirstAlbumYear,
		FirstAlbumYearMax: maxFirstAlbumYear,
		MemberCounts:      memberCounts,
		Countries:         countries,
	}
}

// calculateLocationFilterOptions derives available location filter metadata.
func (s *Store) calculateLocationFilterOptions(locations []Location) LocationFilterOptions {
	if len(locations) == 0 {
		return LocationFilterOptions{}
	}

	minConcerts, maxConcerts := locations[0].TotalConcerts, locations[0].TotalConcerts
	minArtists, maxArtists := locations[0].ArtistCount, locations[0].ArtistCount
	minYear, maxYear := locations[0].EarliestYear, locations[0].LatestYear
	countrySet := make(map[string]bool)

	for _, location := range locations {
		if location.TotalConcerts < minConcerts {
			minConcerts = location.TotalConcerts
		}
		if location.TotalConcerts > maxConcerts {
			maxConcerts = location.TotalConcerts
		}

		if location.ArtistCount < minArtists {
			minArtists = location.ArtistCount
		}
		if location.ArtistCount > maxArtists {
			maxArtists = location.ArtistCount
		}

		if location.EarliestYear > 0 && location.EarliestYear < minYear {
			minYear = location.EarliestYear
		}
		if location.LatestYear > maxYear {
			maxYear = location.LatestYear
		}

		country := location.Country
		if country == "" {
			country = extractCountryFromLocation(location.Name)
		}
		if country != "" {
			countrySet[country] = true
		}
	}

	countries := make([]string, 0, len(countrySet))
	for country := range countrySet {
		countries = append(countries, country)
	}
	sort.Strings(countries)

	return LocationFilterOptions{
		ConcertCountMin: minConcerts,
		ConcertCountMax: maxConcerts,
		ArtistCountMin:  minArtists,
		ArtistCountMax:  maxArtists,
		ConcertYearMin:  minYear,
		ConcertYearMax:  maxYear,
		Countries:       countries,
	}
}
