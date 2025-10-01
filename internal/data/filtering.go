package data

import (
	"strconv"
	"strings"
)

// extractCountryFromLocation parses location strings to extract country names.
func (s *Service) extractCountryFromLocation(location string) string {
	parts := strings.Split(strings.ToLower(location), "-")
	if len(parts) == 0 {
		return ""
	}

	// The country is typically the last part
	country := strings.TrimSpace(parts[len(parts)-1])

	// Handle common abbreviations/normalizations
	switch country {
	case "usa", "us":
		return "USA"
	case "uk":
		return "UK"
	case "uae":
		return "UAE"
	default:
		// Capitalize first letter of each word
		words := strings.Fields(strings.ReplaceAll(country, "-", " "))
		for i, word := range words {
			if len(word) > 0 {
				words[i] = strings.ToUpper(word[:1]) + strings.ToLower(word[1:])
			}
		}
		return strings.Join(words, " ")
	}
}

// extractYearFromDate parses various date string formats to extract years.
func (s *Service) extractYearFromDate(dateStr string) int {
	// Handle common date formats
	if len(dateStr) >= 4 {
		// Check for YYYY at the end (DD-MM-YYYY)
		if len(dateStr) >= 10 && dateStr[2] == '-' && dateStr[5] == '-' {
			if year, err := strconv.Atoi(dateStr[6:10]); err == nil {
				return year
			}
		}
		// Check for YYYY at the beginning (YYYY-MM-DD or just YYYY)
		if year, err := strconv.Atoi(dateStr[:4]); err == nil && year > 1900 && year < 3000 {
			return year
		}
	}
	return 0
}

// --- Artist Filtering System ---

// FilterArtists filters artists based on criteria like creation date, album date, location, and member count.
func (s *Service) FilterArtists(criteria ArtistFilterParams) []Artist {
	artists := s.store.Artists()
	if len(artists) == 0 {
		return nil
	}

	var filtered []Artist
	for _, artist := range artists {
		if s.matchesArtistFilters(artist, criteria) {
			filtered = append(filtered, artist)
		}
	}

	return filtered
}

// matchesArtistFilters checks if an artist matches all specified filter criteria.
func (s *Service) matchesArtistFilters(artist Artist, params ArtistFilterParams) bool {
	// Creation year range filter
	if params.CreationYearFrom != nil && artist.CreationYear < *params.CreationYearFrom {
		return false
	}
	if params.CreationYearTo != nil && artist.CreationYear > *params.CreationYearTo {
		return false
	}

	// First album year filter (extract year from date string)
	if params.FirstAlbumYearFrom != nil || params.FirstAlbumYearTo != nil {
		albumYear := s.extractYearFromDate(artist.FirstAlbum)
		if albumYear > 0 {
			if params.FirstAlbumYearFrom != nil && albumYear < *params.FirstAlbumYearFrom {
				return false
			}
			if params.FirstAlbumYearTo != nil && albumYear > *params.FirstAlbumYearTo {
				return false
			}
		}
	}

	// Member count checkbox filter - check if artist's member count is in the allowed list
	if len(params.MemberCounts) > 0 {
		memberCount := len(artist.Members)
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

	// Country filter - check if artist has concerts in any of the specified countries
	if len(params.Countries) > 0 {
		hasMatchingCountry := false

		// Build allowed set once per filter call for O(1) lookups
		allowed := make(map[string]struct{}, len(params.Countries))
		for _, country := range params.Countries {
			allowed[country] = struct{}{}
		}

		// Check against pre-computed artist.Countries slice
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

// GetArtistFilterOptions returns the precomputed artist filter metadata from the store.
func (s *Service) GetArtistFilterOptions() ArtistFilterOptions {
	return s.store.ArtistFilterOptions()
}

// --- Location Filtering System ---

// FilterLocations filters locations based on concert count, artist count, year range, and country.
func (s *Service) FilterLocations(params LocationFilterParams) []Location {
	locations := s.store.Locations()
	if len(locations) == 0 {
		return nil
	}

	var filtered []Location
	for _, location := range locations {
		if s.matchesLocationFilters(location, params) {
			filtered = append(filtered, location)
		}
	}

	return filtered
}

// matchesLocationFilters checks if a location matches all specified filter criteria.
func (s *Service) matchesLocationFilters(location Location, params LocationFilterParams) bool {
	// Concert count range filter
	if params.ConcertCountFrom != nil && location.TotalConcerts < *params.ConcertCountFrom {
		return false
	}
	if params.ConcertCountTo != nil && location.TotalConcerts > *params.ConcertCountTo {
		return false
	}

	// Artist count range filter
	if params.ArtistCountFrom != nil && location.ArtistCount < *params.ArtistCountFrom {
		return false
	}
	if params.ArtistCountTo != nil && location.ArtistCount > *params.ArtistCountTo {
		return false
	}

	// Concert year range filter
	if params.ConcertYearFrom != nil && location.LatestYear < *params.ConcertYearFrom {
		return false
	}
	if params.ConcertYearTo != nil && location.EarliestYear > *params.ConcertYearTo {
		return false
	}

	// Country filter - check if location's country is in the allowed list
	if len(params.Countries) > 0 {
		locationCountry := s.extractCountryFromLocation(location.Name)
		found := false
		for _, allowedCountry := range params.Countries {
			if locationCountry == allowedCountry {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	return true
}

// GetLocationFilterOptions returns the precomputed location filter metadata from the store.
func (s *Service) GetLocationFilterOptions() LocationFilterOptions {
	return s.store.LocationFilterOptions()
}
