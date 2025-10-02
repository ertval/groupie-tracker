package data

import "sort"

// FilterArtists filters artists based on criteria like creation date, album date, location, and member count.
func (s *Store) FilterArtists(criteria ArtistFilterParams) []Artist {
	artists := s.Artists()
	if len(artists) == 0 {
		return nil
	}

	var filtered []Artist
	for _, artist := range artists {
		if matchesArtistFilters(artist, criteria) {
			filtered = append(filtered, artist)
		}
	}

	return filtered
}

// FilterLocations filters locations based on concert count, artist count, year range, and country.
func (s *Store) FilterLocations(params LocationFilterParams) []Location {
	locations := s.Locations()
	if len(locations) == 0 {
		return nil
	}

	var filtered []Location
	for _, location := range locations {
		if matchesLocationFilters(location, params) {
			filtered = append(filtered, location)
		}
	}

	return filtered
}

// matchesArtistFilters checks if an artist matches all specified filter criteria.
func matchesArtistFilters(artist Artist, params ArtistFilterParams) bool {
	if params.CreationYearFrom != nil && artist.CreationYear < *params.CreationYearFrom {
		return false
	}
	if params.CreationYearTo != nil && artist.CreationYear > *params.CreationYearTo {
		return false
	}

	if params.FirstAlbumYearFrom != nil || params.FirstAlbumYearTo != nil {
		albumYear := artist.FirstAlbumYear
		if albumYear > 0 {
			if params.FirstAlbumYearFrom != nil && albumYear < *params.FirstAlbumYearFrom {
				return false
			}
			if params.FirstAlbumYearTo != nil && albumYear > *params.FirstAlbumYearTo {
				return false
			}
		}
	}

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

	if len(params.Countries) > 0 {
		allowed := make(map[string]struct{}, len(params.Countries))
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

	return true
}

// matchesLocationFilters checks if a location matches all specified filter criteria.
func matchesLocationFilters(location Location, params LocationFilterParams) bool {
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
