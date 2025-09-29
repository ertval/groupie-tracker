package data

import (
	"sort"
	"strconv"
	"strings"
)

// --- Search Data Structures ---

// SearchParams combines search query with optional filters.
type SearchParams struct {
	Query   string  `form:"q" json:"query"`
	Filters Filters `form:"filters" json:"filters"`
}

// SearchResult contains search results and metadata.
type SearchResult struct {
	Artists      []Artist `json:"artists"`
	TotalResults int      `json:"total_results"`
	Query        string   `json:"query"`
}

// SearchSuggestionType categorizes different types of search suggestions.
type SearchSuggestionType string

const (
	SuggestionTypeArtist     SearchSuggestionType = "artist"
	SuggestionTypeMember     SearchSuggestionType = "member"
	SuggestionTypeLocation   SearchSuggestionType = "location"
	SuggestionTypeFirstAlbum SearchSuggestionType = "first-album"
	SuggestionTypeCreation   SearchSuggestionType = "creation"
)

// SearchSuggestion represents a search autocomplete suggestion.
type SearchSuggestion struct {
	Text        string               `json:"text"`
	Type        SearchSuggestionType `json:"type"`
	Description string               `json:"description"`
	URL         string               `json:"url"`
	ArtistID    int                  `json:"artist_id,omitempty"`
}

// --- Core Search Functions ---

// SearchArtists performs comprehensive search across all artist data.
// Searches across artist names, band members, concert locations, creation years, and first album dates.
func SearchArtists(artists []Artist, params SearchParams) SearchResult {
	query := normalizeQuery(params.Query)
	var matchingArtists []Artist

	// If no query provided, use all artists
	if query == "" {
		matchingArtists = artists
	} else {
		// Filter artists by search query
		for _, artist := range artists {
			if matchesSearchQuery(artist, query) {
				matchingArtists = append(matchingArtists, artist)
			}
		}
	}

	// Apply additional filters if provided
	if !params.Filters.IsEmpty() {
		matchingArtists = FilterArtists(matchingArtists, params.Filters)
	}

	return SearchResult{
		Artists:      matchingArtists,
		TotalResults: len(matchingArtists),
		Query:        params.Query,
	}
}

// GenerateSearchSuggestions creates all search suggestions for autocomplete.
// This is expensive to generate but cached at startup for performance.
func GenerateSearchSuggestions(artists []Artist) []SearchSuggestion {
	var suggestions []SearchSuggestion
	seenTexts := make(map[string]bool) // Deduplicate suggestions

	for _, artist := range artists {
		// Artist name suggestions
		if !seenTexts[artist.Name] {
			suggestions = append(suggestions, SearchSuggestion{
				Text:        artist.Name,
				Type:        SuggestionTypeArtist,
				Description: "Artist",
				URL:         "/artists/" + artist.Slug,
				ArtistID:    artist.ID,
			})
			seenTexts[artist.Name] = true
		}

		// Member name suggestions
		for _, member := range artist.Members {
			memberKey := strings.ToLower(member)
			if !seenTexts[memberKey] {
				suggestions = append(suggestions, SearchSuggestion{
					Text:        member,
					Type:        SuggestionTypeMember,
					Description: "Band member of " + artist.Name,
					URL:         "/artists/" + artist.Slug,
					ArtistID:    artist.ID,
				})
				seenTexts[memberKey] = true
			}
		}

		// Location suggestions
		for _, country := range artist.Countries {
			countryKey := strings.ToLower(country)
			if !seenTexts[countryKey] {
				suggestions = append(suggestions, SearchSuggestion{
					Text:        country,
					Type:        SuggestionTypeLocation,
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
				displayLocation := formatLocationName(concert.Location)
				suggestions = append(suggestions, SearchSuggestion{
					Text:        displayLocation,
					Type:        SuggestionTypeLocation,
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
				suggestions = append(suggestions, SearchSuggestion{
					Text:        yearKey,
					Type:        SuggestionTypeCreation,
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
				suggestions = append(suggestions, SearchSuggestion{
					Text:        artist.FirstAlbum,
					Type:        SuggestionTypeFirstAlbum,
					Description: "First album date",
					URL:         "/artists/" + artist.Slug,
					ArtistID:    artist.ID,
				})
				seenTexts[albumKey] = true
			}
		}
	}

	// Sort suggestions by type then text
	sort.Slice(suggestions, func(i, j int) bool {
		if suggestions[i].Type != suggestions[j].Type {
			return suggestions[i].Type < suggestions[j].Type
		}
		return suggestions[i].Text < suggestions[j].Text
	})

	return suggestions
}

// FilterSuggestions filters suggestions based on query for autocomplete.
func FilterSuggestions(suggestions []SearchSuggestion, query string) []SearchSuggestion {
	if query == "" {
		return suggestions
	}

	normalizedQuery := normalizeQuery(query)
	var filtered []SearchSuggestion

	for _, suggestion := range suggestions {
		normalizedText := normalizeQuery(suggestion.Text)
		if strings.Contains(normalizedText, normalizedQuery) {
			filtered = append(filtered, suggestion)
		}
	}

	return filtered
}

// --- Search Matching Logic ---

// matchesSearchQuery checks if an artist matches the search query.
func matchesSearchQuery(artist Artist, normalizedQuery string) bool {
	// Search in artist name
	if strings.Contains(normalizeQuery(artist.Name), normalizedQuery) {
		return true
	}

	// Search in member names
	for _, member := range artist.Members {
		if strings.Contains(normalizeQuery(member), normalizedQuery) {
			return true
		}
	}

	// Search in concert locations
	for _, concert := range artist.Concerts {
		if strings.Contains(normalizeQuery(concert.Location), normalizedQuery) {
			return true
		}
		if strings.Contains(normalizeQuery(concert.Country), normalizedQuery) {
			return true
		}
	}

	// Search in creation year
	if strings.Contains(strconv.Itoa(artist.CreationYear), normalizedQuery) {
		return true
	}

	// Search in first album
	if strings.Contains(normalizeQuery(artist.FirstAlbum), normalizedQuery) {
		return true
	}

	return false
}

// --- Helper Functions ---

// normalizeQuery converts query to lowercase and trims whitespace for consistent searching.
func normalizeQuery(query string) string {
	return strings.ToLower(strings.TrimSpace(query))
}

// formatLocationName converts location slugs to display format.
func formatLocationName(location string) string {
	// Convert "new-york-usa" to "New York USA"
	parts := strings.Split(location, "-")
	for i, part := range parts {
		if len(part) > 0 {
			parts[i] = strings.ToUpper(part[:1]) + strings.ToLower(part[1:])
		}
	}
	return strings.Join(parts, " ")
}
