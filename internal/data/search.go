package data

import (
	"sort"
	"strconv"
	"strings"
)

// GenerateSearchSuggestions creates a list of search suggestions based on user input.
//
// This method performs case-insensitive matching across all searchable data types:
// - Artist names
// - Band member names
// - Concert locations
// - Creation years
// - First album dates
//
// The suggestions are categorized by type to help users understand what they're searching for.
// Results are limited and sorted by relevance to avoid overwhelming the user interface.
func (r *Repository) GenerateSearchSuggestions(query string) []SearchSuggestion {
	query = normalizeSearchQuery(query)
	if query == "" {
		return []SearchSuggestion{}
	}

	var suggestions []SearchSuggestion
	seenSuggestions := make(map[string]bool) // Avoid duplicate suggestions

	// Search artist names
	for _, artist := range r.artists {
		normalizedName := normalizeSearchQuery(artist.Name)
		if strings.Contains(normalizedName, query) {
			suggestionKey := "artist:" + artist.Name
			if !seenSuggestions[suggestionKey] {
				suggestions = append(suggestions, SearchSuggestion{
					Text:        artist.Name,
					Type:        SuggestionTypeArtist,
					Description: artist.Name + " - artist",
					URL:         "/artists/" + artist.Slug,
					ArtistID:    artist.ID,
				})
				seenSuggestions[suggestionKey] = true
			}
		}

		// Search member names
		for _, member := range artist.Members {
			normalizedMember := normalizeSearchQuery(member)
			if strings.Contains(normalizedMember, query) {
				suggestionKey := "member:" + member + ":" + artist.Name
				if !seenSuggestions[suggestionKey] {
					suggestions = append(suggestions, SearchSuggestion{
						Text:        member,
						Type:        SuggestionTypeMember,
						Description: member + " - member of " + artist.Name,
						URL:         "/artists/" + artist.Slug,
						ArtistID:    artist.ID,
					})
					seenSuggestions[suggestionKey] = true
				}
			}
		}

		// Search creation years
		creationYear := strconv.Itoa(artist.CreationYear)
		if strings.Contains(creationYear, query) {
			suggestionKey := "creation:" + creationYear
			if !seenSuggestions[suggestionKey] {
				suggestions = append(suggestions, SearchSuggestion{
					Text:        creationYear,
					Type:        SuggestionTypeCreation,
					Description: creationYear + " - creation date",
					URL:         "/artists?creation=" + creationYear,
					ArtistID:    0,
				})
				seenSuggestions[suggestionKey] = true
			}
		}

		// Search first album dates
		if strings.Contains(artist.FirstAlbum, query) {
			suggestionKey := "album:" + artist.FirstAlbum
			if !seenSuggestions[suggestionKey] {
				suggestions = append(suggestions, SearchSuggestion{
					Text:        artist.FirstAlbum,
					Type:        SuggestionTypeFirstAlbum,
					Description: artist.FirstAlbum + " - first album date",
					URL:         "/artists?album=" + artist.FirstAlbum,
					ArtistID:    0,
				})
				seenSuggestions[suggestionKey] = true
			}
		}
	}

	// Search locations
	for _, location := range r.locations {
		normalizedLocation := normalizeSearchQuery(location.Name)
		if strings.Contains(normalizedLocation, query) {
			suggestionKey := "location:" + location.Name
			if !seenSuggestions[suggestionKey] {
				suggestions = append(suggestions, SearchSuggestion{
					Text:        location.Name,
					Type:        SuggestionTypeLocation,
					Description: location.Name + " - location",
					URL:         "/locations/" + location.Slug,
					ArtistID:    0,
				})
				seenSuggestions[suggestionKey] = true
			}
		}
	}

	// Sort suggestions by type and text for consistent ordering
	sort.Slice(suggestions, func(i, j int) bool {
		if suggestions[i].Type != suggestions[j].Type {
			return suggestions[i].Type < suggestions[j].Type
		}
		return suggestions[i].Text < suggestions[j].Text
	})

	// Limit suggestions to prevent overwhelming the UI
	maxSuggestions := 10
	if len(suggestions) > maxSuggestions {
		suggestions = suggestions[:maxSuggestions]
	}

	return suggestions
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
			suggestions = append(suggestions, SearchSuggestion{
				Text:        artist.Name + " - artist",
				Type:        SuggestionTypeArtist,
				Description: artist.Name + " - artist",
				URL:         "/artists/" + artist.Slug,
				ArtistID:    artist.ID,
			})
			seenSuggestions[artistKey] = true
		}

		// Add member suggestions
		for _, member := range artist.Members {
			memberKey := "member:" + member
			if !seenSuggestions[memberKey] {
				suggestions = append(suggestions, SearchSuggestion{
					Text:        member + " - member",
					Type:        SuggestionTypeMember,
					Description: member + " - member of " + artist.Name,
					URL:         "/artists/" + artist.Slug,
					ArtistID:    artist.ID,
				})
				seenSuggestions[memberKey] = true
			}
		}

		// Add location suggestions
		for location := range artist.DatesAtLocation {
			locationKey := "location:" + location
			if !seenSuggestions[locationKey] {
				suggestions = append(suggestions, SearchSuggestion{
					Text:        location + " - location",
					Type:        SuggestionTypeLocation,
					Description: location + " - concert location",
					URL:         "/search?q=" + location,
					ArtistID:    0, // Not specific to one artist
				})
				seenSuggestions[locationKey] = true
			}
		}

		// Add creation year suggestion
		creationYearStr := strconv.Itoa(artist.CreationYear)
		yearKey := "creation:" + creationYearStr
		if !seenSuggestions[yearKey] {
			suggestions = append(suggestions, SearchSuggestion{
				Text:        creationYearStr + " - creation year",
				Type:        SuggestionTypeCreation,
				Description: "Artists created in " + creationYearStr,
				URL:         "/search?q=" + creationYearStr,
				ArtistID:    0,
			})
			seenSuggestions[yearKey] = true
		}

		// Add first album date suggestion
		albumKey := "album:" + artist.FirstAlbum
		if !seenSuggestions[albumKey] {
			suggestions = append(suggestions, SearchSuggestion{
				Text:        artist.FirstAlbum + " - first album",
				Type:        SuggestionTypeFirstAlbum,
				Description: "Albums released on " + artist.FirstAlbum,
				URL:         "/search?q=" + artist.FirstAlbum,
				ArtistID:    0,
			})
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
