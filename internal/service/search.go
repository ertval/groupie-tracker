package service

import (
	"strconv"
	"strings"

	"groupie-tracker/internal/data"
)

// SearchArtists performs search across artist data with optional filtering.
func (s *Service) SearchArtists(params data.SearchParams) data.SearchResult {
	artists := s.store.Artists()
	normalizedQuery := normalizeSearchQuery(params.Query)
	filtersEmpty := isEmptyFilter(params.Filters)
	useCache := normalizedQuery != "" && filtersEmpty

	if useCache {
		if cached, ok := s.getCachedSearchResults(normalizedQuery); ok {
			return data.SearchResult{
				Artists:      cached,
				Query:        params.Query,
				TotalResults: len(cached),
			}
		}
	}

	var matchingArtists []data.Artist

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
		var filtered []data.Artist
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

	return data.SearchResult{
		Artists:      matchingArtists,
		Query:        params.Query,
		TotalResults: len(matchingArtists),
	}
}

// GenerateAllSearchSuggestions returns the precomputed suggestion cache.
func (s *Service) GenerateAllSearchSuggestions() []data.SearchSuggestion {
	return s.store.Suggestions()
}

func normalizeSearchQuery(query string) string {
	return strings.ToLower(strings.TrimSpace(query))
}

func matchesSearchQuery(artist data.Artist, normalizedQuery string) bool {
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

func isEmptyFilter(filters data.ArtistFilterParams) bool {
	return filters.CreationYearFrom == nil &&
		filters.CreationYearTo == nil &&
		filters.FirstAlbumYearFrom == nil &&
		filters.FirstAlbumYearTo == nil &&
		len(filters.MemberCounts) == 0 &&
		len(filters.Countries) == 0
}

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
