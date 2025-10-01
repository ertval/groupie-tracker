package domain

import (
	"sort"
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

// SearchArtists performs comprehensive search across all artist data.
//
// This method combines text search with optional filtering to provide
// powerful search capabilities. It searches across:
// - Artist names (case-insensitive)
// - Band member names (case-insensitive)
// - Concert locations (case-insensitive)
// - Creation years (exact match)
// - First album dates (substring match)
//
// The search can be combined with existing filter parameters for
// refined results (e.g., "Phil Collins created after 1980").
func (r *Repository) SearchArtists(params SearchParams) SearchResult {
	query := normalizeSearchQuery(params.Query)
	var matchingArtists []Artist

	// If no query provided, use all artists
	if query == "" {
		matchingArtists = r.artists
	} else {
		// Filter artists by search query
		for _, artist := range r.artists {
			if matchesSearchQuery(artist, query) {
				matchingArtists = append(matchingArtists, artist)
			}
		}
	}

	// Apply additional filters if provided
	if !isEmptyFilter(params.Filters) {
		var filteredArtists []Artist
		for _, artist := range matchingArtists {
			if r.matchesArtistFilters(artist, params.Filters) {
				filteredArtists = append(filteredArtists, artist)
			}
		}
		matchingArtists = filteredArtists
	}

	return SearchResult{
		Artists:      matchingArtists,
		Query:        params.Query,
		TotalResults: len(matchingArtists),
	}
}

// normalizeSearchQuery standardizes search input for consistent matching.
//
// This function handles common search input variations:
// - Converts to lowercase for case-insensitive search
// - Trims whitespace from beginning and end
// - Preserves internal spaces and special characters
//
// The normalization ensures that "Queen", "QUEEN", and " Queen " all
// produce identical search results.
func normalizeSearchQuery(query string) string {
	return strings.ToLower(strings.TrimSpace(query))
}

// matchesSearchQuery checks if an artist matches the given search query.
//
// This function implements the core search logic by checking if the query
// appears in any of the artist's searchable fields:
// - Artist name
// - Any band member name
// - Creation year (as string)
// - First album date (substring match)
// - Any country where they performed
//
// All text matching is case-insensitive via normalizeSearchQuery.
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

// isEmptyFilter checks if filter parameters are empty/unset.
//
// This helper function determines whether any filter criteria have been
// specified, allowing the search to skip filter processing when no
// additional filtering is needed.
func isEmptyFilter(filters ArtistFilterParams) bool {
	return filters.CreationYearFrom == nil &&
		filters.CreationYearTo == nil &&
		filters.FirstAlbumYearFrom == nil &&
		filters.FirstAlbumYearTo == nil &&
		len(filters.MemberCounts) == 0 &&
		len(filters.Countries) == 0
}

// GenerateAllSearchSuggestions creates a comprehensive list of all possible search suggestions
// for use in HTML datalist elements. This provides all available search options upfront
// for client-side autocomplete without requiring JavaScript.
func (r *Repository) GenerateAllSearchSuggestions() []SearchSuggestion {
	var suggestions []SearchSuggestion
	seenSuggestions := make(map[string]bool)

	for _, artist := range r.artists {
		// Add artist name suggestion
		artistKey := "artist:" + artist.Name
		if !seenSuggestions[artistKey] {
			suggestions = append(suggestions, newSearchSuggestion(
				artist.Name+" - artist",
				string(SuggestionTypeArtist),
				artist.Name+" - artist",
				"/artists/"+artist.Slug,
				artist.ID,
			))
			seenSuggestions[artistKey] = true
		}

		// Add member suggestions
		for _, member := range artist.Members {
			memberKey := "member:" + member
			if !seenSuggestions[memberKey] {
				suggestions = append(suggestions, newSearchSuggestion(
					member+" - member",
					string(SuggestionTypeMember),
					member+" - member of "+artist.Name,
					"/artists/"+artist.Slug,
					artist.ID,
				))
				seenSuggestions[memberKey] = true
			}
		}

		// Add location suggestions
		for location := range artist.DatesAtLocation {
			locationKey := "location:" + location
			if !seenSuggestions[locationKey] {
				suggestions = append(suggestions, newSearchSuggestion(
					location+" - location",
					string(SuggestionTypeLocation),
					location+" - concert location",
					"/search?q="+location,
					0, // Not specific to one artist
				))
				seenSuggestions[locationKey] = true
			}
		}

		// Add creation year suggestion
		creationYearStr := strconv.Itoa(artist.CreationYear)
		yearKey := "creation:" + creationYearStr
		if !seenSuggestions[yearKey] {
			suggestions = append(suggestions, newSearchSuggestion(
				creationYearStr+" - creation year",
				string(SuggestionTypeCreation),
				"Artists created in "+creationYearStr,
				"/search?q="+creationYearStr,
				0,
			))
			seenSuggestions[yearKey] = true
		}

		// Add first album date suggestion
		albumKey := "album:" + artist.FirstAlbum
		if !seenSuggestions[albumKey] {
			suggestions = append(suggestions, newSearchSuggestion(
				artist.FirstAlbum+" - first album",
				string(SuggestionTypeFirstAlbum),
				"Albums released on "+artist.FirstAlbum,
				"/search?q="+artist.FirstAlbum,
				0,
			))
			seenSuggestions[albumKey] = true
		}
	}

	// Sort suggestions by type and then by text for consistent ordering
	sort.Slice(suggestions, func(i, j int) bool {
		if suggestions[i].Type != suggestions[j].Type {
			return suggestions[i].Type < suggestions[j].Type
		}
		return suggestions[i].Text < suggestions[j].Text
	})

	return suggestions
}

// locationMatches checks if a location name matches a search query using various formats.
//
// This function handles multiple location search patterns for locations in "city-country" format:
// - Direct match: "london-uk" matches query "london"
// - Country match: "london-uk" matches query "uk"
// - Hyphenated match: "london-uk" matches query "london-uk"
// - Space match: "london-uk" matches query "london uk" (converts spaces to hyphens)
// - Partial city match: "new-york-usa" matches query "new york"
//
// All matching is case-insensitive to provide user-friendly search experience.
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

// FilterSuggestionsOptimized provides optimized suggestion filtering with prioritization and limits.
//
// This function implements several performance optimizations:
// 1. Uses pre-computed normalized text to avoid repeated string.ToLower() calls
// 2. Implements match prioritization (exact > prefix > contains)
// 3. Limits results to prevent overwhelming the UI
// 4. Stops early when enough results are found
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
