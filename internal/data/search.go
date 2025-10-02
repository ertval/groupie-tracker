package data

import (
	"strconv"
	"strings"
)

// newSearchSuggestion creates a SearchSuggestion with normalized text for efficient filtering
func newSearchSuggestion(text, suggestionType, description, url string, artistID int) SearchSuggestion {
	return SearchSuggestion{
		Text:           text,
		Type:           SearchSuggestionType(suggestionType),
		Description:    description,
		URL:            url,
		ArtistID:       artistID,
		normalizedText: strings.ToLower(text),
	}
}

// SearchArtists performs search across artist data with optional filtering.
func (s *Service) SearchArtists(params SearchParams) SearchResult {
	artists := s.store.Artists()
	normalizedQuery := normalizeSearchQuery(params.Query)
	filtersEmpty := isEmptyFilter(params.Filters)
	useCache := normalizedQuery != "" && filtersEmpty

	if useCache {
		if cached, ok := s.getCachedSearchResults(normalizedQuery); ok {
			return SearchResult{
				Artists:      cached,
				Query:        params.Query,
				TotalResults: len(cached),
			}
		}
	}

	var matchingArtists []Artist

	// If no query provided, use all artists
	if normalizedQuery == "" {
		matchingArtists = artists
	} else {
		// Filter artists by search query
		for _, artist := range artists {
			if matchesSearchQuery(artist, normalizedQuery) {
				matchingArtists = append(matchingArtists, artist)
			}
		}
	}

	// Apply additional filters if provided
	if !filtersEmpty {
		var filteredArtists []Artist
		for _, artist := range matchingArtists {
			if s.matchesArtistFilters(artist, params.Filters) {
				filteredArtists = append(filteredArtists, artist)
			}
		}
		matchingArtists = filteredArtists
	}

	if useCache {
		s.setCachedSearchResults(normalizedQuery, matchingArtists)
	}

	return SearchResult{
		Artists:      matchingArtists,
		Query:        params.Query,
		TotalResults: len(matchingArtists),
	}
}

// normalizeSearchQuery converts query to lowercase and trims whitespace.
func normalizeSearchQuery(query string) string {
	return strings.ToLower(strings.TrimSpace(query))
}

// matchesSearchQuery checks if an artist matches the search query in any field.
func matchesSearchQuery(artist Artist, normalizedQuery string) bool {
	if normalizedQuery == "" {
		return true
	}

	// Check artist name
	if strings.Contains(normalizeSearchQuery(artist.Name), normalizedQuery) {
		return true
	}

	// Check member names
	for _, member := range artist.Members {
		if strings.Contains(normalizeSearchQuery(member), normalizedQuery) {
			return true
		}
	}

	// Check creation year
	creationYear := strconv.Itoa(artist.CreationYear)
	if strings.Contains(creationYear, normalizedQuery) {
		return true
	}

	// Check first album date
	if strings.Contains(strings.ToLower(artist.FirstAlbum), normalizedQuery) {
		return true
	}

	// Check countries
	for _, country := range artist.Countries {
		if strings.Contains(normalizeSearchQuery(country), normalizedQuery) {
			return true
		}
	}

	// Check concert locations (cities and full location names)
	for _, concert := range artist.Concerts {
		if locationMatches(concert.Location, normalizedQuery) {
			return true
		}
	}

	return false
}

// isEmptyFilter checks if filter parameters are empty.
func isEmptyFilter(filters ArtistFilterParams) bool {
	return filters.CreationYearFrom == nil &&
		filters.CreationYearTo == nil &&
		filters.FirstAlbumYearFrom == nil &&
		filters.FirstAlbumYearTo == nil &&
		len(filters.MemberCounts) == 0 &&
		len(filters.Countries) == 0
}

// GenerateAllSearchSuggestions returns the precomputed suggestion cache.
func (s *Service) GenerateAllSearchSuggestions() []SearchSuggestion {
	return s.store.Suggestions()
}

// locationMatches checks if a location name matches a search query.
func locationMatches(locationName, query string) bool {
	normalizedLocation := normalizeSearchQuery(locationName)
	normalizedQuery := normalizeSearchQuery(query)

	// Direct substring match (e.g., "london" matches "london-uk")
	if strings.Contains(normalizedLocation, normalizedQuery) {
		return true
	}

	// Convert spaces in query to hyphens for matching (e.g., "london uk" -> "london-uk")
	hyphenatedQuery := strings.ReplaceAll(normalizedQuery, " ", "-")
	if normalizedLocation == hyphenatedQuery {
		return true
	}

	// Parse location into parts by splitting on hyphens
	parts := strings.Split(locationName, "-") // Split "london-uk" or "new-york-usa"
	if len(parts) < 2 {
		return false // Location doesn't have expected format
	}

	// Extract country (last part) and city (everything else joined)
	country := parts[len(parts)-1]
	city := strings.Join(parts[:len(parts)-1], "-") // "new-york" from "new-york-usa"

	normalizedCity := normalizeSearchQuery(city)
	normalizedCountry := normalizeSearchQuery(country)

	// Check for individual city or country match
	if normalizedQuery == normalizedCity || normalizedQuery == normalizedCountry {
		return true
	}

	// Check for space-separated city match (e.g., "new york" matches "new-york")
	cityWithSpaces := strings.ReplaceAll(normalizedCity, "-", " ")
	if normalizedQuery == cityWithSpaces {
		return true
	}

	return false
}

// FilterSuggestionsOptimized filters and prioritizes search suggestions.
func FilterSuggestionsOptimized(suggestions []SearchSuggestion, query string, maxResults int) []SearchSuggestion {
	if query == "" || len(suggestions) == 0 {
		return []SearchSuggestion{}
	}

	if maxResults <= 0 {
		maxResults = 20 // Default reasonable limit
	}

	queryLower := strings.ToLower(strings.TrimSpace(query))

	// Three tiers of matches for prioritization
	var exactMatches []SearchSuggestion
	var prefixMatches []SearchSuggestion
	var containsMatches []SearchSuggestion

	totalFound := 0

	for _, suggestion := range suggestions {
		if totalFound >= maxResults {
			break // Early termination once we have enough results
		}

		// Use pre-computed normalized text for efficient comparison
		normalizedText := suggestion.normalizedText

		if normalizedText == queryLower {
			// Exact match - highest priority
			exactMatches = append(exactMatches, suggestion)
			totalFound++
		} else if strings.HasPrefix(normalizedText, queryLower) {
			// Prefix match - medium priority
			prefixMatches = append(prefixMatches, suggestion)
			totalFound++
		} else if strings.Contains(normalizedText, queryLower) {
			// Contains match - lowest priority
			containsMatches = append(containsMatches, suggestion)
			totalFound++
		}
	}

	// Combine results in priority order
	var results []SearchSuggestion
	results = append(results, exactMatches...)
	results = append(results, prefixMatches...)
	results = append(results, containsMatches...)

	// Ensure we don't exceed the limit
	if len(results) > maxResults {
		results = results[:maxResults]
	}

	return results
}
