package data

import (
	"sort"
	"strconv"
	"strings"
)

// --- Geographic and Date Parsing Utilities ---

// extractCountryFromLocation parses location strings to extract country names.
//
// Handles common location formats used in the Groupie Tracker API:
//   - "city-country" (e.g., "london-uk")
//   - "city-state-country" (e.g., "new-york-usa")
//   - "city-region-country" (e.g., "abu-dhabi-united-arab-emirates")
//
// The function normalizes country abbreviations (USA, UK, UAE) and applies
// proper title casing to multi-word country names. This ensures consistent
// country identification for filtering operations.
//
// Returns the normalized country name, or empty string if parsing fails.
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

// extractYearFromDate parses various date string formats to extract calendar years.
//
// Supports common date formats from the Groupie Tracker API:
//   - "DD-MM-YYYY" format (26-03-2001)
//   - "YYYY-MM-DD" format (2001-03-26)
//   - "YYYY" format (2001)
//
// This function enables filtering by album release years and concert years
// regardless of the original API date format. Year values are validated to be
// within reasonable bounds (1900-3000) to prevent parsing errors.
//
// Returns the extracted year as integer, or 0 if no valid year is found.
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

// --- Artist Filtering System ---

// FilterArtists applies multi-criteria filtering to the complete artist collection.
//
// This is the primary artist filtering function that accepts comprehensive filter
// parameters and returns matching artists. Supports the following filter types:
//   - Creation year range: Band formation years
//   - First album year range: Debut album release years
//   - Member count selection: Number of band members (checkbox filtering)
//   - Country selection: Countries where artists have performed concerts
//
// All filters are applied with AND logic - artists must match ALL active criteria
// to be included in results. Empty filter parameters are ignored (no filtering applied).
//
// Returns a slice of artists that match all specified filter criteria.
func (r *Repository) FilterArtists(params ArtistFilterParams) []*Artist {
	var filtered []*Artist

	for _, artist := range r.artists {
		if r.matchesArtistFilters(*artist, params) {
			filtered = append(filtered, artist)
		}
	}

	return filtered
}

// matchesArtistFilters determines if a single artist satisfies all active filter criteria.
//
// This helper function implements the core filtering logic by testing each artist
// against all provided filter parameters:
//
// Creation Year Filtering:
//   - Tests artist formation year against optional min/max bounds
//   - Uses inclusive range matching (>= min, <= max)
//
// First Album Year Filtering:
//   - Extracts year from artist's first album date string
//   - Applies same inclusive range logic as creation year
//   - Skips filtering if album date cannot be parsed
//
// Member Count Filtering:
//   - Counts current band members from artist.Members slice
//   - Checks if count appears in the allowed member counts list
//   - Uses exact matching (not range-based)
//
// Country Filtering:
//   - Extracts countries from all artist concert locations
//   - Checks if any artist country matches filter country list
//   - Uses string-based country matching after normalization
//
// Returns true only if artist passes ALL active filters (AND logic).
func (r *Repository) matchesArtistFilters(artist Artist, params ArtistFilterParams) bool {
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

// GetArtistFilterOptions analyzes current artist data to compute filter bounds and available options.
//
// This function scans the complete artist collection to determine:
//
// Range Bounds:
//   - Min/max creation years across all artists (for range sliders)
//   - Min/max first album years extracted from date strings
//   - Fallback to creation year bounds if album dates are unparseable
//
// Discrete Options:
//   - Unique member counts (1 to maximum found, sorted ascending)
//   - Unique countries extracted from all concert locations (sorted alphabetically)
//
// The computed options are used by the frontend to configure filter UI elements:
// range sliders get min/max bounds, checkboxes get available option lists.
//
// This ensures filter options always reflect the actual data distribution,
// preventing empty result sets from impossible filter combinations.
//
// Returns structured filter options ready for template rendering and JSON APIs.
func (r *Repository) GetArtistFilterOptions() ArtistFilterOptions {
	if len(r.artists) == 0 {
		return ArtistFilterOptions{}
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

		// Collect unique countries from pre-computed Countries field
		for _, country := range artist.Countries {
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

	return ArtistFilterOptions{
		CreationYearMin:   minCreationYear,
		CreationYearMax:   maxCreationYear,
		FirstAlbumYearMin: minFirstAlbumYear,
		FirstAlbumYearMax: maxFirstAlbumYear,
		MemberCounts:      memberCounts,
		Countries:         countries,
	}
}

// --- Location Filtering System ---

// FilterLocations applies multi-criteria filtering to the complete location collection.
//
// This function mirrors the artist filtering approach but operates on concert locations.
// Supports location-specific filter criteria:
//   - Concert count range: Total concerts held at location
//   - Artist count range: Number of unique artists who performed there
//   - Concert year range: Date range of concerts at location
//   - Country selection: Geographic country filtering
//
// Like artist filtering, all criteria use AND logic - locations must satisfy
// ALL active filters to appear in results.
//
// Returns a slice of locations matching all specified filter criteria.
func (r *Repository) FilterLocations(params LocationFilterParams) []*Location {
	var filtered []*Location

	for _, location := range r.locations {
		if r.matchesLocationFilters(*location, params) {
			filtered = append(filtered, location)
		}
	}

	return filtered
}

// matchesLocationFilters determines if a single location satisfies all active filter criteria.
//
// This helper implements location-specific filtering logic by testing each location
// against provided filter parameters:
//
// Concert Count Filtering:
//   - Tests location.TotalConcerts against optional min/max bounds
//   - Uses inclusive range matching for concert volume filtering
//
// Artist Count Filtering:
//   - Tests location.ArtistCount (unique artists) against range bounds
//   - Helps find venues that host many different artists vs. few regulars
//
// Concert Year Filtering:
//   - Uses location.EarliestYear and location.LatestYear for range checks
//   - Enables temporal filtering (e.g., venues active in specific decades)
//   - Year bounds are pre-computed during location creation
//
// Country Filtering:
//   - Extracts country from location name using standard parsing
//   - Matches against allowed country list with exact string comparison
//   - Uses same normalization as artist country filtering
//
// Returns true only if location passes ALL active filters (AND logic).
func (r *Repository) matchesLocationFilters(location Location, params LocationFilterParams) bool {
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
		locationCountry := r.extractCountryFromLocation(location.Name)
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

// GetLocationFilterOptions analyzes current location data to compute filter bounds and available options.
//
// This function scans the complete location collection to determine:
//
// Range Bounds:
//   - Min/max concert counts across all locations (for volume filtering)
//   - Min/max artist counts (for venue diversity filtering)
//   - Min/max concert years (for temporal filtering)
//   - All bounds use inclusive ranges for intuitive slider behavior
//
// Discrete Options:
//   - Unique countries extracted from location names (sorted alphabetically)
//   - Same country normalization as used in artist filtering for consistency
//
// The computed options configure location filter UI elements and ensure that
// filter ranges reflect actual data distribution. This prevents users from
// setting impossible filter combinations that would return no results.
//
// Year bounds use the pre-computed EarliestYear/LatestYear fields from each
// location, which are calculated during the location creation process.
//
// Returns structured location filter options ready for template rendering and JSON APIs.
func (r *Repository) GetLocationFilterOptions() LocationFilterOptions {
	if len(r.locations) == 0 {
		return LocationFilterOptions{}
	}

	// Initialize with first location's values
	minConcerts, maxConcerts := r.locations[0].TotalConcerts, r.locations[0].TotalConcerts
	minArtists, maxArtists := r.locations[0].ArtistCount, r.locations[0].ArtistCount
	minYear, maxYear := r.locations[0].EarliestYear, r.locations[0].LatestYear
	countrySet := make(map[string]bool)

	for _, location := range r.locations {
		// Concert count range
		if location.TotalConcerts < minConcerts {
			minConcerts = location.TotalConcerts
		}
		if location.TotalConcerts > maxConcerts {
			maxConcerts = location.TotalConcerts
		}

		// Artist count range
		if location.ArtistCount < minArtists {
			minArtists = location.ArtistCount
		}
		if location.ArtistCount > maxArtists {
			maxArtists = location.ArtistCount
		}

		// Concert year range
		if location.EarliestYear > 0 && location.EarliestYear < minYear {
			minYear = location.EarliestYear
		}
		if location.LatestYear > maxYear {
			maxYear = location.LatestYear
		}

		// Collect unique countries
		country := r.extractCountryFromLocation(location.Name)
		if country != "" {
			countrySet[country] = true
		}
	}

	// Convert country set to sorted slice
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
