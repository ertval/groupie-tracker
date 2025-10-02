package data

import (
	"sort"
	"strconv"
	"strings"
)

// SearchArtists performs search across artist data with optional filtering.
func (s *Store) SearchArtists(params SearchParams) SearchResult {
	artists := s.Artists()
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

	if normalizedQuery == "" {
		matchingArtists = artists
	} else {
		for _, artist := range artists {
			if matchesSearchQuery(artist, normalizedQuery) {
				matchingArtists = append(matchingArtists, artist)
			}
		}
	}

	if !filtersEmpty {
		var filtered []Artist
		for _, artist := range matchingArtists {
			if matchesArtistFilters(artist, params.Filters) {
				filtered = append(filtered, artist)
			}
		}
		matchingArtists = filtered
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

// GenerateAllSearchSuggestions returns the precomputed suggestion cache.
func (s *Store) GenerateAllSearchSuggestions() []SearchSuggestion {
	return s.Suggestions()
}

// FilterSearchSuggestions returns suggestions matching the query ordered by relevance.
func (s *Store) FilterSearchSuggestions(query string, maxResults int) []SearchSuggestion {
	suggestions := s.Suggestions()
	return filterSearchSuggestions(suggestions, query, maxResults)
}

// GetAdjacentArtists finds the previous and next artists in alphabetical order.
func (s *Store) GetAdjacentArtists(currentID int) (prev, next *Artist) {
	index, ok := s.ArtistPosition(currentID)
	if !ok {
		return nil, nil
	}

	artists := s.Artists()
	if len(artists) == 0 {
		return nil, nil
	}

	if index > 0 {
		prev = &artists[index-1]
	}

	if index < len(artists)-1 {
		next = &artists[index+1]
	}

	return prev, next
}

// normalizeSearchQuery converts query to lowercase and trims whitespace.
func normalizeSearchQuery(query string) string {
	return strings.ToLower(strings.TrimSpace(query))
}

// filterSearchSuggestions filters and ranks suggestions based on the query.
func filterSearchSuggestions(suggestions []SearchSuggestion, query string, maxResults int) []SearchSuggestion {
	normalizedQuery := normalizeSearchQuery(query)
	if normalizedQuery == "" || len(suggestions) == 0 {
		return []SearchSuggestion{}
	}

	if maxResults <= 0 {
		maxResults = 20
	}

	var exactMatches []SearchSuggestion
	var prefixMatches []SearchSuggestion
	var containsMatches []SearchSuggestion

	totalFound := 0

	for _, suggestion := range suggestions {
		if totalFound >= maxResults {
			break
		}

		normalizedText := normalizeSearchQuery(suggestion.Text)

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

	results := make([]SearchSuggestion, 0, len(exactMatches)+len(prefixMatches)+len(containsMatches))
	results = append(results, exactMatches...)
	results = append(results, prefixMatches...)
	results = append(results, containsMatches...)

	if len(results) > maxResults {
		results = results[:maxResults]
	}

	return results
}

// matchesSearchQuery checks if an artist matches the search query.
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

func (s *Store) generateSearchSuggestions(artists []Artist) []SearchSuggestion {
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
