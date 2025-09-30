package service

import (
	"sort"
	"strconv"
	"strings"

	"groupie-tracker/internal/models"
)

// FilterService handles filtering functionality for artists.
type FilterService struct{}

// NewFilterService creates a new filter service.
func NewFilterService() *FilterService {
	return &FilterService{}
}

// FilterArtists applies multi-criteria filtering to artists.
// All filters use AND logic - artists must match ALL active criteria.
func (fs *FilterService) FilterArtists(artists []models.Artist, filters models.Filters) []models.Artist {
	var filtered []models.Artist

	for _, artist := range artists {
		if fs.matchesFilters(artist, filters) {
			filtered = append(filtered, artist)
		}
	}

	return filtered
}

// GetFilterOptions computes available filter options from the dataset.
// This is computed on-demand since it's a cheap operation with small dataset.
func (fs *FilterService) GetFilterOptions(artists []models.Artist) models.FilterOptions {
	if len(artists) == 0 {
		return models.FilterOptions{}
	}

	// Initialize with first artist values
	options := models.FilterOptions{
		CreationYearMin:   artists[0].CreationYear,
		CreationYearMax:   artists[0].CreationYear,
		FirstAlbumYearMin: 9999,
		FirstAlbumYearMax: 0,
	}

	memberCountSet := make(map[int]bool)
	countrySet := make(map[string]bool)

	// Single pass to collect all available options
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
		if albumYear := fs.extractYearFromAlbum(artist.FirstAlbum); albumYear > 0 {
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

// ParseFiltersFromForm extracts filter parameters from HTTP form data.
func (fs *FilterService) ParseFiltersFromForm(formValues map[string][]string) models.Filters {
	filters := models.Filters{}

	// Parse year ranges
	if val := fs.getFormValue(formValues, "creation_year_min"); val != "" {
		filters.CreationYearMin = fs.parseInt(val)
	}
	if val := fs.getFormValue(formValues, "creation_year_max"); val != "" {
		filters.CreationYearMax = fs.parseInt(val)
	}
	if val := fs.getFormValue(formValues, "first_album_year_min"); val != "" {
		filters.FirstAlbumYearMin = fs.parseInt(val)
	}
	if val := fs.getFormValue(formValues, "first_album_year_max"); val != "" {
		filters.FirstAlbumYearMax = fs.parseInt(val)
	}

	// Parse member counts (checkbox array)
	if values, exists := formValues["member_counts"]; exists {
		for _, val := range values {
			if count := fs.parseInt(val); count > 0 {
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

// matchesFilters checks if an artist matches all active filter criteria.
func (fs *FilterService) matchesFilters(artist models.Artist, filters models.Filters) bool {
	// Creation year range filter
	if filters.CreationYearMin > 0 && artist.CreationYear < filters.CreationYearMin {
		return false
	}
	if filters.CreationYearMax > 0 && artist.CreationYear > filters.CreationYearMax {
		return false
	}

	// First album year range filter
	albumYear := fs.extractYearFromAlbum(artist.FirstAlbum)
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
func (fs *FilterService) extractYearFromAlbum(albumDate string) int {
	return fs.extractYearFromDate(albumDate)
}

// extractYearFromDate parses various date formats to extract calendar years.
// Duplicated from DataService for independence - consider moving to a shared util package.
func (fs *FilterService) extractYearFromDate(dateStr string) int {
	if len(dateStr) < 4 {
		return 0
	}

	// Handle DD-MM-YYYY format
	if len(dateStr) >= 10 && dateStr[2] == '-' && dateStr[5] == '-' {
		if year := fs.parseYear(dateStr[6:10]); year > 0 {
			return year
		}
	}

	// Handle YYYY-MM-DD format or just YYYY
	if year := fs.parseYear(dateStr[:4]); year > 0 {
		return year
	}

	return 0
}

// parseYear safely parses a 4-digit year string with validation.
func (fs *FilterService) parseYear(yearStr string) int {
	if len(yearStr) != 4 {
		return 0
	}

	if year, err := strconv.Atoi(yearStr); err == nil {
		// Reasonable year range validation
		if year >= 1900 && year <= 2100 {
			return year
		}
	}

	return 0
}

// getFormValue safely extracts the first value for a form field.
func (fs *FilterService) getFormValue(values map[string][]string, key string) string {
	if vals, exists := values[key]; exists && len(vals) > 0 {
		return strings.TrimSpace(vals[0])
	}
	return ""
}

// parseInt safely parses an integer, returning 0 on error.
func (fs *FilterService) parseInt(s string) int {
	if i, err := strconv.Atoi(s); err == nil && i > 0 {
		return i
	}
	return 0
}
