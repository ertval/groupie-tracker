package service

import (
	"sort"
	"strconv"
	"strings"

	"groupie-tracker/internal/models"
)

// SearchService handles search functionality and suggestion generation.
type SearchService struct{}

// NewSearchService creates a new search service.
func NewSearchService() *SearchService {
	return &SearchService{}
}

// Search performs comprehensive search across all artist data.
// Searches across artist names, band members, concert locations, creation years, and first album dates.
func (ss *SearchService) Search(artists []models.Artist, params models.SearchParams) models.SearchResult {
	query := ss.normalizeQuery(params.Query)
	var matchingArtists []models.Artist

	// If no query provided, use all artists
	if query == "" {
		matchingArtists = artists
	} else {
		// Filter artists by search query
		for _, artist := range artists {
			if ss.matchesSearchQuery(artist, query) {
				matchingArtists = append(matchingArtists, artist)
			}
		}
	}

	// Apply additional filters if provided
	if !params.Filters.IsEmpty() {
		filterService := NewFilterService()
		matchingArtists = filterService.FilterArtists(matchingArtists, params.Filters)
	}

	return models.SearchResult{
		Artists:      matchingArtists,
		TotalResults: len(matchingArtists),
		Query:        params.Query,
	}
}

// GenerateSuggestions creates comprehensive search suggestions for autocomplete functionality.
// This is expensive to generate but should be cached at startup for performance.
func (ss *SearchService) GenerateSuggestions(artists []models.Artist) []models.SearchSuggestion {
	var suggestions []models.SearchSuggestion
	seenTexts := make(map[string]bool) // Deduplicate suggestions

	for _, artist := range artists {
		// Artist name suggestions
		if !seenTexts[strings.ToLower(artist.Name)] {
			suggestions = append(suggestions, models.SearchSuggestion{
				Text:        artist.Name,
				Type:        models.SuggestionTypeArtist,
				Description: "Artist",
				URL:         "/artists/" + artist.Slug,
				ArtistID:    artist.ID,
			})
			seenTexts[strings.ToLower(artist.Name)] = true
		}

		// Member name suggestions
		for _, member := range artist.Members {
			memberKey := strings.ToLower(member)
			if !seenTexts[memberKey] {
				suggestions = append(suggestions, models.SearchSuggestion{
					Text:        member,
					Type:        models.SuggestionTypeMember,
					Description: "Band member of " + artist.Name,
					URL:         "/artists/" + artist.Slug,
					ArtistID:    artist.ID,
				})
				seenTexts[memberKey] = true
			}
		}

		// Location suggestions - countries
		for _, country := range artist.Countries {
			countryKey := strings.ToLower(country)
			if !seenTexts[countryKey] {
				suggestions = append(suggestions, models.SearchSuggestion{
					Text:        country,
					Type:        models.SuggestionTypeLocation,
					Description: "Concert location",
					URL:         "/search?q=" + country,
				})
				seenTexts[countryKey] = true
			}
		}

		// Unique concert locations
		for _, concert := range artist.Concerts {
			locationKey := strings.ToLower(concert.Location)
			if !seenTexts[locationKey] && concert.Location != "" {
				displayLocation := ss.formatLocationName(concert.Location)
				suggestions = append(suggestions, models.SearchSuggestion{
					Text:        displayLocation,
					Type:        models.SuggestionTypeLocation,
					Description: "Concert venue",
					URL:         "/search?q=" + concert.Location,
				})
				seenTexts[locationKey] = true
			}
		}

		// Creation year suggestions
		if artist.CreationYear > 0 {
			yearKey := strconv.Itoa(artist.CreationYear)
			if !seenTexts[yearKey] {
				suggestions = append(suggestions, models.SearchSuggestion{
					Text:        yearKey,
					Type:        models.SuggestionTypeCreation,
					Description: "Formation year",
					URL:         "/search?q=" + yearKey,
				})
				seenTexts[yearKey] = true
			}
		}

		// First album suggestions
		if artist.FirstAlbum != "" {
			albumKey := strings.ToLower(artist.FirstAlbum)
			if !seenTexts[albumKey] {
				suggestions = append(suggestions, models.SearchSuggestion{
					Text:        artist.FirstAlbum,
					Type:        models.SuggestionTypeFirstAlbum,
					Description: "First album date",
					URL:         "/artists/" + artist.Slug,
					ArtistID:    artist.ID,
				})
				seenTexts[albumKey] = true
			}
		}
	}

	// Sort suggestions by type then text for consistent presentation
	sort.Slice(suggestions, func(i, j int) bool {
		if suggestions[i].Type != suggestions[j].Type {
			return suggestions[i].Type < suggestions[j].Type
		}
		return suggestions[i].Text < suggestions[j].Text
	})

	return suggestions
}

// FilterSuggestions filters suggestions based on query for autocomplete.
func (ss *SearchService) FilterSuggestions(suggestions []models.SearchSuggestion, query string) []models.SearchSuggestion {
	if query == "" {
		return suggestions
	}

	normalizedQuery := ss.normalizeQuery(query)
	var filtered []models.SearchSuggestion

	for _, suggestion := range suggestions {
		normalizedText := ss.normalizeQuery(suggestion.Text)
		if strings.Contains(normalizedText, normalizedQuery) {
			filtered = append(filtered, suggestion)
		}
	}

	return filtered
}

// matchesSearchQuery checks if an artist matches the search query across all searchable fields.
func (ss *SearchService) matchesSearchQuery(artist models.Artist, normalizedQuery string) bool {
	// Search in artist name
	if strings.Contains(ss.normalizeQuery(artist.Name), normalizedQuery) {
		return true
	}

	// Search in member names
	for _, member := range artist.Members {
		if strings.Contains(ss.normalizeQuery(member), normalizedQuery) {
			return true
		}
	}

	// Search in concert locations and countries
	for _, concert := range artist.Concerts {
		if strings.Contains(ss.normalizeQuery(concert.Location), normalizedQuery) {
			return true
		}
		if strings.Contains(ss.normalizeQuery(concert.Country), normalizedQuery) {
			return true
		}
	}

	// Search in creation year
	if strings.Contains(strconv.Itoa(artist.CreationYear), normalizedQuery) {
		return true
	}

	// Search in first album
	if strings.Contains(ss.normalizeQuery(artist.FirstAlbum), normalizedQuery) {
		return true
	}

	return false
}

// normalizeQuery converts query to lowercase and trims whitespace for consistent searching.
func (ss *SearchService) normalizeQuery(query string) string {
	return strings.ToLower(strings.TrimSpace(query))
}

// formatLocationName converts location strings to display format.
func (ss *SearchService) formatLocationName(location string) string {
	// Convert "new-york-usa" to "New York USA"
	parts := strings.Split(location, "-")
	for i, part := range parts {
		if len(part) > 0 {
			parts[i] = strings.ToUpper(part[:1]) + strings.ToLower(part[1:])
		}
	}
	return strings.Join(parts, " ")
}
