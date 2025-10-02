package data

import (
	"sort"
	"strconv"
	"strings"
)

// SearchArtists performs full-text search across artist names, members, and metadata with optional filtering.
// Uses LRU caching for repeated identical queries (when filters are empty) to improve performance.
// Search matches: artist name, member names, creation year, first album year, and location names.
func (s *Store) SearchArtists(params SearchParams) SearchResult {
	artists := s.Artists()
	normalizedQuery := normalizeSearchQuery(params.Query) // Lowercase and trim for case-insensitive matching
	filtersEmpty := isEmptyFilter(params.Filters)
	useCache := normalizedQuery != "" && filtersEmpty // Only cache pure search queries without filters

	// Try to retrieve from cache if applicable (skip cache if filters are active)
	if useCache {
		if cached, ok := s.getCachedSearchResults(normalizedQuery); ok {
			return SearchResult{
				Artists:      cached,
				Query:        params.Query,
				TotalResults: len(cached),
			}
		}
	}

	var matchingArtists []*Artist

	// If query is empty, return all artists (useful for filter-only operations)
	if normalizedQuery == "" {
		matchingArtists = artists
	} else {
		// Linear search through all artists (acceptable since dataset is small ~52 artists)
		for _, artist := range artists {
			if matchesSearchQuery(*artist, normalizedQuery) { // Check name, members, years, locations
				matchingArtists = append(matchingArtists, artist)
			}
		}
	}

	// Apply filters if any are specified (filters are ANDed with search results)
	if !filtersEmpty {
		var filtered []*Artist
		for _, artist := range matchingArtists {
			if matchesArtistFilters(*artist, params.Filters) { // Reuse filter logic from filters.go
				filtered = append(filtered, artist)
			}
		}
		matchingArtists = filtered
	}

	// Store in cache for future identical queries (only if using cache and filters are empty)
	if useCache {
		s.setCachedSearchResults(normalizedQuery, matchingArtists)
	}

	return SearchResult{
		Artists:      matchingArtists,
		Query:        params.Query,
		TotalResults: len(matchingArtists),
	}
}

// GenerateAllSearchSuggestions returns the complete precomputed suggestion list for autocomplete.
// Suggestions are generated at startup and include artist names, member names, and location names.
func (s *Store) GenerateAllSearchSuggestions() []SearchSuggestion {
	return s.Suggestions()
}

// FilterSearchSuggestions returns suggestions matching the query, ordered by relevance (exact → prefix → contains).
// Limits results to maxResults (default 20) to prevent overwhelming the autocomplete UI.
func (s *Store) FilterSearchSuggestions(query string, maxResults int) []SearchSuggestion {
	suggestions := s.Suggestions()
	return filterSearchSuggestions(suggestions, query, maxResults)
}

// GetAdjacentArtists finds the previous and next artists relative to the current artist in alphabetical order.
// Used for "Previous Artist" and "Next Artist" navigation links in the UI.
// Returns nil for prev if at beginning, nil for next if at end.
func (s *Store) GetAdjacentArtists(currentID int) (prev, next *Artist) {
	index, ok := s.ArtistPosition(currentID) // Get current artist's position in sorted slice
	if !ok {
		return nil, nil // Artist ID not found
	}

	artists := s.Artists()
	if len(artists) == 0 {
		return nil, nil
	}

	// Get previous artist if not at beginning (index > 0)
	if index > 0 {
		prev = artists[index-1]
	}

	// Get next artist if not at end (index < len-1)
	if index < len(artists)-1 {
		next = artists[index+1]
	}

	return prev, next
}

// normalizeSearchQuery standardizes query strings for case-insensitive comparison.
// Converts to lowercase and trims whitespace to ensure "QUEEN" matches "queen".
func normalizeSearchQuery(query string) string {
	return strings.ToLower(strings.TrimSpace(query))
}

// filterSearchSuggestions filters and ranks suggestions based on query relevance.
// Ranking priority: exact matches first, then prefix matches, then contains matches.
// This provides better UX by showing most relevant suggestions at the top.
func filterSearchSuggestions(suggestions []SearchSuggestion, query string, maxResults int) []SearchSuggestion {
	normalizedQuery := normalizeSearchQuery(query)
	if normalizedQuery == "" || len(suggestions) == 0 {
		return []SearchSuggestion{} // Empty query returns no suggestions
	}

	if maxResults <= 0 {
		maxResults = 20 // Default limit prevents excessive autocomplete results
	}

	var exactMatches []SearchSuggestion    // "queen" matches "queen" exactly
	var prefixMatches []SearchSuggestion   // "qu" matches "queen" by prefix
	var containsMatches []SearchSuggestion // "ee" matches "queen" by substring

	totalFound := 0 // Track total matches to stop early once we hit maxResults

	for _, suggestion := range suggestions {
		if totalFound >= maxResults {
			break // Stop searching once we have enough results
		}

		normalizedText := normalizeSearchQuery(suggestion.Text)

		// Categorize by match type (order matters for priority)
		switch {
		case normalizedText == normalizedQuery:
			exactMatches = append(exactMatches, suggestion)
			totalFound++
		case strings.HasPrefix(normalizedText, normalizedQuery):
			prefixMatches = append(prefixMatches, suggestion)
			totalFound++
		case strings.Contains(normalizedText, normalizedQuery):
			containsMatches = append(containsMatches, suggestion)
			totalFound++
		}
	}

	// Combine results in order of relevance: exact → prefix → contains
	results := make([]SearchSuggestion, 0, len(exactMatches)+len(prefixMatches)+len(containsMatches))
	results = append(results, exactMatches...)
	results = append(results, prefixMatches...)
	results = append(results, containsMatches...)

	// Enforce maxResults limit (in case we collected more due to parallel categorization)
	if len(results) > maxResults {
		results = results[:maxResults]
	}

	return results
}

// matchesSearchQuery determines if an artist matches the search query by checking multiple fields.
// Checks: artist name, member names, creation year, first album year, and concert location names.
// Returns true if ANY field contains the query (OR logic across fields).
func matchesSearchQuery(artist Artist, normalizedQuery string) bool {
	if normalizedQuery == "" {
		return true
	}

	if strings.Contains(strings.ToLower(artist.Name), normalizedQuery) {
		return true
	}

	for _, member := range artist.Members {
		if strings.Contains(strings.ToLower(member), normalizedQuery) {
			return true
		}
	}

	creationYear := strconv.Itoa(artist.CreationYear)
	if strings.Contains(creationYear, normalizedQuery) {
		return true
	}

	if strings.Contains(strings.ToLower(artist.FirstAlbum), normalizedQuery) {
		return true
	}

	for _, country := range artist.Countries {
		if strings.Contains(strings.ToLower(country), normalizedQuery) {
			return true
		}
	}

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

// locationMatches checks if a location matches the query.
func locationMatches(locationName, query string) bool {
	normalizedLocation := normalizeSearchQuery(locationName)
	normalizedQuery := normalizeSearchQuery(query)

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

	normalizedCity := normalizeSearchQuery(city)
	normalizedCountry := normalizeSearchQuery(country)

	if normalizedQuery == normalizedCity || normalizedQuery == normalizedCountry {
		return true
	}

	cityWithSpaces := strings.ReplaceAll(normalizedCity, "-", " ")
	return normalizedQuery == cityWithSpaces
}

// generateSearchSuggestions pre-computes autocomplete suggestions from the dataset.
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

func (s *Store) generateSearchSuggestions(artists []*Artist) []SearchSuggestion {
	var suggestions []SearchSuggestion
	seen := make(map[string]bool)

	for _, artist := range artists {
		artistKey := "artist:" + artist.Name
		if !seen[artistKey] {
			suggestions = append(suggestions, newSearchSuggestion(
				artist.Name+" - artist",
				string(SuggestionTypeArtist),
				artist.Name+" - artist",
				"/artists/"+artist.Slug,
				artist.ID,
			))
			seen[artistKey] = true
		}

		for _, member := range artist.Members {
			memberKey := "member:" + member
			if !seen[memberKey] {
				suggestions = append(suggestions, newSearchSuggestion(
					member+" - member",
					string(SuggestionTypeMember),
					member+" - member of "+artist.Name,
					"/artists/"+artist.Slug,
					artist.ID,
				))
				seen[memberKey] = true
			}
		}

		for location := range artist.DatesAtLocation {
			locationKey := "location:" + location
			if !seen[locationKey] {
				suggestions = append(suggestions, newSearchSuggestion(
					location+" - location",
					string(SuggestionTypeLocation),
					location+" - concert location",
					"/search?q="+location,
					0,
				))
				seen[locationKey] = true
			}
		}

		creationYear := strconv.Itoa(artist.CreationYear)
		yearKey := "creation:" + creationYear
		if !seen[yearKey] {
			suggestions = append(suggestions, newSearchSuggestion(
				creationYear+" - creation year",
				string(SuggestionTypeCreation),
				"Artists created in "+creationYear,
				"/search?q="+creationYear,
				0,
			))
			seen[yearKey] = true
		}

		albumKey := "album:" + artist.FirstAlbum
		if !seen[albumKey] {
			suggestions = append(suggestions, newSearchSuggestion(
				artist.FirstAlbum+" - first album",
				string(SuggestionTypeFirstAlbum),
				"Albums released on "+artist.FirstAlbum,
				"/search?q="+artist.FirstAlbum,
				0,
			))
			seen[albumKey] = true
		}
	}

	sort.Slice(suggestions, func(i, j int) bool {
		if suggestions[i].Type != suggestions[j].Type {
			return suggestions[i].Type < suggestions[j].Type
		}
		return suggestions[i].Text < suggestions[j].Text
	})

	return suggestions
}
