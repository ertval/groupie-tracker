package service

import "groupie-tracker/internal/data"

// FilterArtists filters artists based on criteria like creation date, album date, location, and member count.
func (s *Service) FilterArtists(criteria data.ArtistFilterParams) []data.Artist {
	artists := s.store.Artists()
	if len(artists) == 0 {
		return nil
	}

	var filtered []data.Artist
	for _, artist := range artists {
		if matchesArtistFilters(artist, criteria) {
			filtered = append(filtered, artist)
		}
	}

	return filtered
}

// GetArtistFilterOptions returns the precomputed artist filter metadata from the store.
func (s *Service) GetArtistFilterOptions() data.ArtistFilterOptions {
	return s.store.ArtistFilterOptions()
}

// FilterLocations filters locations based on concert count, artist count, year range, and country.
func (s *Service) FilterLocations(params data.LocationFilterParams) []data.Location {
	locations := s.store.Locations()
	if len(locations) == 0 {
		return nil
	}

	var filtered []data.Location
	for _, location := range locations {
		if matchesLocationFilters(location, params) {
			filtered = append(filtered, location)
		}
	}

	return filtered
}

// GetLocationFilterOptions returns the precomputed location filter metadata from the store.
func (s *Service) GetLocationFilterOptions() data.LocationFilterOptions {
	return s.store.LocationFilterOptions()
}

// matchesArtistFilters checks if an artist matches all specified filter criteria.
func matchesArtistFilters(artist data.Artist, params data.ArtistFilterParams) bool {
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
func matchesLocationFilters(location data.Location, params data.LocationFilterParams) bool {
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
