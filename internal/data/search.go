package data

import "strings"

// newSearchSuggestion creates a SearchSuggestion with normalized text for efficient filtering.
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

// FilterSuggestionsOptimized filters and prioritizes search suggestions based on the query.
func FilterSuggestionsOptimized(suggestions []SearchSuggestion, query string, maxResults int) []SearchSuggestion {
	if query == "" || len(suggestions) == 0 {
		return []SearchSuggestion{}
	}

	if maxResults <= 0 {
		maxResults = 20
	}

	queryLower := strings.ToLower(strings.TrimSpace(query))

	var exactMatches []SearchSuggestion
	var prefixMatches []SearchSuggestion
	var containsMatches []SearchSuggestion

	totalFound := 0

	for _, suggestion := range suggestions {
		if totalFound >= maxResults {
			break
		}

		normalizedText := suggestion.normalizedText

		switch {
		case normalizedText == queryLower:
			exactMatches = append(exactMatches, suggestion)
			totalFound++
		case strings.HasPrefix(normalizedText, queryLower):
			prefixMatches = append(prefixMatches, suggestion)
			totalFound++
		case strings.Contains(normalizedText, queryLower):
			containsMatches = append(containsMatches, suggestion)
			totalFound++
		}
	}

	results := make([]SearchSuggestion, 0, len(exactMatches)+len(prefixMatches)+len(containsMatches))
	results = append(results, exactMatches...)
	results = append(results, prefixMatches...)
	results = append(results, containsMatches...)

	if len(results) > maxResults {
		results = results[:maxResults]
	}

	return results
}

// Helper retained for tests that relied on location matching logic in the data package.
func locationMatches(locationName, query string) bool {
	normalizedLocation := strings.ToLower(strings.TrimSpace(locationName))
	normalizedQuery := strings.ToLower(strings.TrimSpace(query))

	if strings.Contains(normalizedLocation, normalizedQuery) {
		return true
	}

	hyphenatedQuery := strings.ReplaceAll(normalizedQuery, " ", "-")
	if normalizedLocation == hyphenatedQuery {
		return true
	}

	parts := strings.Split(locationName, "-")
	if len(parts) < 2 {
		return false
	}

	country := parts[len(parts)-1]
	city := strings.Join(parts[:len(parts)-1], "-")

	normalizedCity := strings.ToLower(strings.TrimSpace(city))
	normalizedCountry := strings.ToLower(strings.TrimSpace(country))

	if normalizedQuery == normalizedCity || normalizedQuery == normalizedCountry {
		return true
	}

	cityWithSpaces := strings.ReplaceAll(normalizedCity, "-", " ")
	return normalizedQuery == cityWithSpaces
}
