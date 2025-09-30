// Package service contains business logic for data processing, search, and filtering.
// This layer orchestrates data transformation and provides the core functionality
// for the application.
package service

import (
	"regexp"
	"sort"
	"strconv"
	"strings"

	"groupie-tracker/internal/api"
	"groupie-tracker/internal/models"
)

// DataService handles data processing and transformation from API to domain models.
type DataService struct{}

// NewDataService creates a new data processing service.
func NewDataService() *DataService {
	return &DataService{}
}

// ProcessAPIData converts API data to domain models with full data enrichment.
// This is a single-pass processing function that optimizes for performance.
func (ds *DataService) ProcessAPIData(apiArtists []api.APIArtist, apiRelations []api.APIRelationIndex) ([]models.Artist, []models.Location, models.AppStats) {
	// Create relation map for fast lookup
	relationMap := make(map[int]api.APIRelationIndex, len(apiRelations))
	for _, relation := range apiRelations {
		relationMap[relation.ID] = relation
	}

	// Process artists with concert data
	artists := ds.processArtists(apiArtists, relationMap)

	// Create locations from artist data
	locations := ds.createLocations(artists)

	// Calculate comprehensive statistics
	stats := ds.calculateStats(artists, locations)

	return artists, locations, stats
}

// processArtists converts API data to enriched domain models.
func (ds *DataService) processArtists(apiArtists []api.APIArtist, relationMap map[int]api.APIRelationIndex) []models.Artist {
	artists := make([]models.Artist, 0, len(apiArtists))

	for _, apiArtist := range apiArtists {
		artist := models.Artist{
			ID:           apiArtist.ID,
			Name:         apiArtist.Name,
			Slug:         ds.generateSlug(apiArtist.Name),
			Members:      apiArtist.Members,
			CreationYear: apiArtist.CreationYear,
			FirstAlbum:   apiArtist.FirstAlbum,
			Image:        apiArtist.Image,
		}

		// Add concert data if available
		if relation, exists := relationMap[apiArtist.ID]; exists {
			concerts, countries := ds.processConcerts(relation.DatesLocations)
			artist.Concerts = concerts
			artist.Countries = countries
			artist.ConcertCount = len(concerts)
		}

		artists = append(artists, artist)
	}

	// Sort artists by name for consistent ordering
	sort.Slice(artists, func(i, j int) bool {
		return artists[i].Name < artists[j].Name
	})

	return artists
}

// processConcerts converts API concert data to domain models with country extraction.
func (ds *DataService) processConcerts(datesLocations map[string][]string) ([]models.Concert, []string) {
	var concerts []models.Concert
	countrySet := make(map[string]bool)

	for location, dates := range datesLocations {
		country := ds.extractCountryFromLocation(location)
		if country != "" {
			countrySet[country] = true
		}

		for _, date := range dates {
			concert := models.Concert{
				Location: location,
				Country:  country,
				Date:     date,
				Year:     ds.extractYearFromDate(date),
			}
			concerts = append(concerts, concert)
		}
	}

	// Convert country set to sorted slice
	countries := make([]string, 0, len(countrySet))
	for country := range countrySet {
		countries = append(countries, country)
	}
	sort.Strings(countries)

	return concerts, countries
}

// createLocations generates location aggregates from artist concert data.
func (ds *DataService) createLocations(artists []models.Artist) []models.Location {
	locationMap := make(map[string]*models.Location)

	// Single pass aggregation by location
	for _, artist := range artists {
		for _, concert := range artist.Concerts {
			if concert.Location == "" {
				continue
			}

			if loc, exists := locationMap[concert.Location]; exists {
				// Update existing location statistics
				ds.updateLocationStats(loc, artist.Name, concert)
			} else {
				// Create new location entry
				location := &models.Location{
					Name:          concert.Location,
					Slug:          ds.generateSlug(concert.Location),
					Artists:       []string{artist.Name},
					ArtistCount:   1,
					TotalConcerts: 1,
					Concerts:      []models.Concert{concert},
				}

				if concert.Year > 0 {
					location.EarliestYear = concert.Year
					location.LatestYear = concert.Year
				}

				locationMap[concert.Location] = location
			}
		}
	}

	// Convert map to slice and sort by concert count (descending)
	locations := make([]models.Location, 0, len(locationMap))
	for _, location := range locationMap {
		locations = append(locations, *location)
	}

	sort.Slice(locations, func(i, j int) bool {
		return locations[i].TotalConcerts > locations[j].TotalConcerts
	})

	return locations
}

// updateLocationStats efficiently updates location statistics with new concert data.
func (ds *DataService) updateLocationStats(loc *models.Location, artistName string, concert models.Concert) {
	loc.TotalConcerts++

	// Update year range
	if concert.Year > 0 {
		if loc.EarliestYear == 0 || concert.Year < loc.EarliestYear {
			loc.EarliestYear = concert.Year
		}
		if concert.Year > loc.LatestYear {
			loc.LatestYear = concert.Year
		}
	}

	// Add artist if not already present (linear search is fine for small lists)
	artistExists := false
	for _, existingArtist := range loc.Artists {
		if existingArtist == artistName {
			artistExists = true
			break
		}
	}
	if !artistExists {
		loc.Artists = append(loc.Artists, artistName)
		loc.ArtistCount++
	}

	loc.Concerts = append(loc.Concerts, concert)
}

// calculateStats computes comprehensive application statistics.
func (ds *DataService) calculateStats(artists []models.Artist, locations []models.Location) models.AppStats {
	stats := models.AppStats{
		TotalArtists:   len(artists),
		TotalLocations: len(locations),
	}

	// Single pass through artists for all statistics
	for _, artist := range artists {
		stats.TotalConcerts += artist.ConcertCount

		// Track earliest/latest years across all concerts
		for _, concert := range artist.Concerts {
			if concert.Year > 0 {
				if stats.EarliestYear == 0 || concert.Year < stats.EarliestYear {
					stats.EarliestYear = concert.Year
				}
				if concert.Year > stats.LatestYear {
					stats.LatestYear = concert.Year
				}
			}
		}
	}

	return stats
}

// generateSlug creates a URL-friendly slug from any string.
func (ds *DataService) generateSlug(s string) string {
	// Convert to lowercase
	slug := strings.ToLower(s)

	// Replace spaces and special characters with hyphens
	reg := regexp.MustCompile(`[^a-z0-9]+`)
	slug = reg.ReplaceAllString(slug, "-")

	// Remove leading/trailing hyphens
	slug = strings.Trim(slug, "-")

	return slug
}

// extractCountryFromLocation parses location strings to extract normalized country names.
func (ds *DataService) extractCountryFromLocation(location string) string {
	parts := strings.Split(strings.ToLower(location), "-")
	if len(parts) == 0 {
		return ""
	}

	// The country is typically the last part
	country := strings.TrimSpace(parts[len(parts)-1])

	// Handle common abbreviations and normalize formatting
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

// extractYearFromDate parses various date formats to extract calendar years.
func (ds *DataService) extractYearFromDate(dateStr string) int {
	if len(dateStr) < 4 {
		return 0
	}

	// Handle DD-MM-YYYY format
	if len(dateStr) >= 10 && dateStr[2] == '-' && dateStr[5] == '-' {
		if year := ds.parseYear(dateStr[6:10]); year > 0 {
			return year
		}
	}

	// Handle YYYY-MM-DD format or just YYYY
	if year := ds.parseYear(dateStr[:4]); year > 0 {
		return year
	}

	return 0
}

// parseYear safely parses a 4-digit year string with validation.
func (ds *DataService) parseYear(yearStr string) int {
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
