package data

import (
	"sort"
	"strconv"
	"strings"
)

// --- Filter Data Structures ---

// Filters represents filter parameters with simple zero values instead of pointers.
type Filters struct {
	CreationYearMin int      `form:"creation_year_min" json:"creation_year_min"`
	CreationYearMax int      `form:"creation_year_max" json:"creation_year_max"`
	FirstAlbumYearMin int    `form:"first_album_year_min" json:"first_album_year_min"`
	FirstAlbumYearMax int    `form:"first_album_year_max" json:"first_album_year_max"`
	MemberCounts    []int    `form:"member_counts" json:"member_counts"`
	Countries       []string `form:"countries" json:"countries"`
}

// FilterOptions provides the available filter bounds and options.
type FilterOptions struct {
	CreationYearMin   int      `json:"creation_year_min"`
	CreationYearMax   int      `json:"creation_year_max"`
	FirstAlbumYearMin int      `json:"first_album_year_min"`
	FirstAlbumYearMax int      `json:"first_album_year_max"`
	MemberCounts      []int    `json:"member_counts"`
	Countries         []string `json:"countries"`
}

// --- Core Filtering Functions ---

// FilterArtists applies multi-criteria filtering to artists.
// All filters use AND logic - artists must match ALL active criteria.
func FilterArtists(artists []Artist, filters Filters) []Artist {
	var filtered []Artist

	for _, artist := range artists {
		if matchesFilters(artist, filters) {
			filtered = append(filtered, artist)
		}
	}

	return filtered
}

// GetFilterOptions computes available filter options from the dataset.
// This is computed on-demand since it's a cheap operation with small dataset.
func GetFilterOptions(artists []Artist) FilterOptions {
	if len(artists) == 0 {
		return FilterOptions{}
	}

	// Initialize with first artist
	options := FilterOptions{
		CreationYearMin:   artists[0].CreationYear,
		CreationYearMax:   artists[0].CreationYear,
		FirstAlbumYearMin: 9999,
		FirstAlbumYearMax: 0,
	}

	memberCountSet := make(map[int]bool)
	countrySet := make(map[string]bool)

	// Collect all available options
	for _, artist := range artists {
		// Track creation year bounds
		if artist.CreationYear > 0 {
			if artist.CreationYear < options.CreationYearMin {
				options.CreationYearMin = artist.CreationYear
			}
			if artist.CreationYear > options.CreationYearMax {
				options.CreationYearMax = artist.CreationYear
			}
		}

		// Track first album year bounds
		if albumYear := extractYearFromAlbum(artist.FirstAlbum); albumYear > 0 {
			if albumYear < options.FirstAlbumYearMin {
				options.FirstAlbumYearMin = albumYear
			}
			if albumYear > options.FirstAlbumYearMax {
				options.FirstAlbumYearMax = albumYear
			}
		}

		// Track member counts
		memberCount := len(artist.Members)
		if memberCount > 0 {
			memberCountSet[memberCount] = true
		}

		// Track countries
		for _, country := range artist.Countries {
			if country != "" {
				countrySet[country] = true
			}
		}
	}

	// Convert sets to sorted slices
	for count := range memberCountSet {
		options.MemberCounts = append(options.MemberCounts, count)
	}
	sort.Ints(options.MemberCounts)

	for country := range countrySet {
		options.Countries = append(options.Countries, country)
	}
	sort.Strings(options.Countries)

	// Handle edge case where no album years found
	if options.FirstAlbumYearMin == 9999 {
		options.FirstAlbumYearMin = options.CreationYearMin
	}

	return options
}

// --- Filter Matching Logic ---

// matchesFilters checks if an artist matches all active filter criteria.
func matchesFilters(artist Artist, filters Filters) bool {
	// Creation year range filter
	if filters.CreationYearMin > 0 && artist.CreationYear < filters.CreationYearMin {
		return false
	}
	if filters.CreationYearMax > 0 && artist.CreationYear > filters.CreationYearMax {
		return false
	}

	// First album year range filter
	albumYear := extractYearFromAlbum(artist.FirstAlbum)
	if filters.FirstAlbumYearMin > 0 && albumYear < filters.FirstAlbumYearMin {
		return false
	}
	if filters.FirstAlbumYearMax > 0 && albumYear > filters.FirstAlbumYearMax {
		return false
	}

	// Member count filter (any of the selected counts)
	if len(filters.MemberCounts) > 0 {
		memberCount := len(artist.Members)
		found := false
		for _, allowedCount := range filters.MemberCounts {
			if memberCount == allowedCount {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// Country filter (artist must have performed in any of the selected countries)
	if len(filters.Countries) > 0 {
		found := false
		for _, filterCountry := range filters.Countries {
			for _, artistCountry := range artist.Countries {
				if strings.EqualFold(artistCountry, filterCountry) {
					found = true
					break
				}
			}
			if found {
				break
			}
		}
		if !found {
			return false
		}
	}

	return true
}

// extractYearFromAlbum extracts year from first album date string.
func extractYearFromAlbum(albumDate string) int {
	return extractYearFromDate(albumDate)
}

// --- Form Parsing Helpers ---

// ParseFiltersFromForm extracts filter parameters from HTTP form data.
func ParseFiltersFromForm(formValues map[string][]string) Filters {
	filters := Filters{}

	// Parse year ranges
	if val := getFormValue(formValues, "creation_year_min"); val != "" {
		filters.CreationYearMin = parseInt(val)
	}
	if val := getFormValue(formValues, "creation_year_max"); val != "" {
		filters.CreationYearMax = parseInt(val)
	}
	if val := getFormValue(formValues, "first_album_year_min"); val != "" {
		filters.FirstAlbumYearMin = parseInt(val)
	}
	if val := getFormValue(formValues, "first_album_year_max"); val != "" {
		filters.FirstAlbumYearMax = parseInt(val)
	}

	// Parse member counts (checkbox array)
	if values, exists := formValues["member_counts"]; exists {
		for _, val := range values {
			if count := parseInt(val); count > 0 {
				filters.MemberCounts = append(filters.MemberCounts, count)
			}
		}
	}

	// Parse countries (checkbox array)
	if values, exists := formValues["countries"]; exists {
		for _, val := range values {
			if val != "" {
				filters.Countries = append(filters.Countries, val)
			}
		}
	}

	return filters
}

// IsEmpty checks if any filters are active.
func (f Filters) IsEmpty() bool {
	return f.CreationYearMin == 0 &&
		f.CreationYearMax == 0 &&
		f.FirstAlbumYearMin == 0 &&
		f.FirstAlbumYearMax == 0 &&
		len(f.MemberCounts) == 0 &&
		len(f.Countries) == 0
}

// --- Helper Functions ---

// getFormValue safely extracts the first value for a form field.
func getFormValue(values map[string][]string, key string) string {
	if vals, exists := values[key]; exists && len(vals) > 0 {
		return strings.TrimSpace(vals[0])
	}
	return ""
}

// parseInt safely parses an integer, returning 0 on error.
func parseInt(s string) int {
	if i, err := strconv.Atoi(s); err == nil && i > 0 {
		return i
	}
	return 0
}