package data

import (
	"sort"
	"strconv"
	"strings"
)

// --- Filter Functionality ---

// extractCountryFromLocation extracts the country from a location string
// Assumes location format like "city-state-country" or "city-country"
func (r *Repository) extractCountryFromLocation(location string) string {
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

// FilterArtists filters the artists based on the provided criteria
func (r *Repository) FilterArtists(params FilterParams) []Artist {
	var filtered []Artist

	for _, artist := range r.artists {
		if r.matchesFilters(artist, params) {
			filtered = append(filtered, artist)
		}
	}

	return filtered
}

// matchesFilters checks if an artist matches the filter criteria
func (r *Repository) matchesFilters(artist Artist, params FilterParams) bool {
	// Creation year range filter
	if params.CreationYearFrom != nil && artist.CreationYear < *params.CreationYearFrom {
		return false
	}
	if params.CreationYearTo != nil && artist.CreationYear > *params.CreationYearTo {
		return false
	}

	// First album year filter (extract year from date string)
	if params.FirstAlbumYearFrom != nil || params.FirstAlbumYearTo != nil {
		albumYear := r.extractYearFromDate(artist.FirstAlbum)
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
		artistCountries := make(map[string]bool)

		// Extract countries from artist's concert locations
		for _, concert := range artist.Concerts {
			country := r.extractCountryFromLocation(concert.Location)
			if country != "" {
				artistCountries[country] = true
			}
		}

		// Check if any artist country matches filter countries
		for _, filterCountry := range params.Countries {
			if artistCountries[filterCountry] {
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

// extractYearFromDate extracts year from various date formats
func (r *Repository) extractYearFromDate(dateStr string) int {
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

// GetFilterOptions returns the available filter options based on current data
func (r *Repository) GetFilterOptions() FilterOptions {
	if len(r.artists) == 0 {
		return FilterOptions{}
	}

	// Initialize with first artist's values
	minCreationYear, maxCreationYear := r.artists[0].CreationYear, r.artists[0].CreationYear
	minFirstAlbumYear, maxFirstAlbumYear := 0, 0
	memberCountSet := make(map[int]bool)
	countrySet := make(map[string]bool)

	for _, artist := range r.artists {
		// Creation year range
		if artist.CreationYear < minCreationYear {
			minCreationYear = artist.CreationYear
		}
		if artist.CreationYear > maxCreationYear {
			maxCreationYear = artist.CreationYear
		}

		// First album year range
		albumYear := r.extractYearFromDate(artist.FirstAlbum)
		if albumYear > 0 {
			if minFirstAlbumYear == 0 || albumYear < minFirstAlbumYear {
				minFirstAlbumYear = albumYear
			}
			if albumYear > maxFirstAlbumYear {
				maxFirstAlbumYear = albumYear
			}
		}

		// Collect unique member counts
		memberCount := len(artist.Members)
		memberCountSet[memberCount] = true

		// Collect unique countries from concert locations
		for _, concert := range artist.Concerts {
			country := r.extractCountryFromLocation(concert.Location)
			if country != "" {
				countrySet[country] = true
			}
		}
	}

	// Convert member count set to sorted slice (1 to max)
	memberCounts := make([]int, 0, len(memberCountSet))
	for count := range memberCountSet {
		memberCounts = append(memberCounts, count)
	}
	sort.Ints(memberCounts)

	// Convert country set to sorted slice
	countries := make([]string, 0, len(countrySet))
	for country := range countrySet {
		countries = append(countries, country)
	}
	sort.Strings(countries)

	// Set default first album year range if no valid years found
	if minFirstAlbumYear == 0 {
		minFirstAlbumYear = minCreationYear
	}
	if maxFirstAlbumYear == 0 {
		maxFirstAlbumYear = maxCreationYear
	}

	return FilterOptions{
		CreationYearMin:   minCreationYear,
		CreationYearMax:   maxCreationYear,
		FirstAlbumYearMin: minFirstAlbumYear,
		FirstAlbumYearMax: maxFirstAlbumYear,
		MemberCounts:      memberCounts,
		Countries:         countries,
	}
}
