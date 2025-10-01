package web

import (
	"fmt"
	"groupie-tracker/internal/domain"
	"strings"
)

// --- Template-Specific Data Structures ---
// These structs contain pre-formatted display data to eliminate business logic from templates.
// All formatting is done server-side to keep templates simple and display-focused.

// TemplateArtist contains pre-formatted artist data for display in templates.
// Eliminates the need for template functions like join, len calculations, and pluralization logic.
type TemplateArtist struct {
	// Raw data fields
	Name         string   `json:"name"`
	Slug         string   `json:"slug"`
	Image        string   `json:"image"`
	CreationYear int      `json:"creation_year"`
	FirstAlbum   string   `json:"first_album"`
	Members      []string `json:"members"`
	ConcertCount int      `json:"concert_count"`
	Countries    []string `json:"countries"`

	// Pre-formatted display fields (eliminates template logic)
	DisplayName      string `json:"display_name"`       // Same as Name but ready for display
	MemberCountText  string `json:"member_count_text"`  // "4 members" or "1 member"
	CountriesText    string `json:"countries_text"`     // "USA, UK, Canada" (pre-joined)
	ConcertCountText string `json:"concert_count_text"` // "12 concerts" or "1 concert"
	MembersText      string `json:"members_text"`       // "John, Paul, George, Ringo" (pre-joined)
	CreationText     string `json:"creation_text"`      // "Founded 1960" or "Created 1960"
}

// TemplateLocation contains pre-formatted location data for display.
type TemplateLocation struct {
	// Raw data fields
	Name          string `json:"name"`
	Slug          string `json:"slug"`
	ArtistCount   int    `json:"artist_count"`
	TotalConcerts int    `json:"total_concerts"`
	EarliestYear  int    `json:"earliest_year"`
	LatestYear    int    `json:"latest_year"`

	// Pre-formatted display fields
	DisplayName      string `json:"display_name"`       // Title-cased name: "new-york" -> "New York"
	ArtistCountText  string `json:"artist_count_text"`  // "5 artists" or "1 artist"
	ConcertCountText string `json:"concert_count_text"` // "12 concerts" or "1 concert"
	YearRangeText    string `json:"year_range_text"`    // "1970 - 1990" or "2020"
}

// TemplateFilterOptions contains pre-formatted filter option domain.
type TemplateFilterOptions struct {
	// Raw data
	CreationYearMin   int      `json:"creation_year_min"`
	CreationYearMax   int      `json:"creation_year_max"`
	FirstAlbumYearMin int      `json:"first_album_year_min"`
	FirstAlbumYearMax int      `json:"first_album_year_max"`
	MemberCounts      []int    `json:"member_counts"`
	Countries         []string `json:"countries"`

	// Pre-formatted display fields
	CreationYearRange   string   `json:"creation_year_range"`    // "1960 - 2020"
	FirstAlbumYearRange string   `json:"first_album_year_range"` // "1962 - 2021"
	MemberCountOptions  []string `json:"member_count_options"`   // ["1 member", "2 members", ...]
}

// TemplateAppliedFilters contains pre-formatted applied filter data for display.
type TemplateAppliedFilters struct {
	// Raw filter data
	CreationYearFrom   *int     `json:"creation_year_from"`
	CreationYearTo     *int     `json:"creation_year_to"`
	FirstAlbumYearFrom *int     `json:"first_album_year_from"`
	FirstAlbumYearTo   *int     `json:"first_album_year_to"`
	MemberCounts       []int    `json:"member_counts"`
	Countries          []string `json:"countries"`

	// Pre-formatted display fields
	CreationYearText   string `json:"creation_year_text"`    // "1970 - 1980" or empty
	FirstAlbumYearText string `json:"first_album_year_text"` // "1975 - 1985" or empty
	MemberCountsText   string `json:"member_counts_text"`    // "1, 2, 4 members" or empty
	CountriesText      string `json:"countries_text"`        // "USA, UK" or empty
	IsActive           bool   `json:"is_active"`             // True if any filters applied
}

// TemplateSearchResult contains pre-formatted search result domain.
type TemplateSearchResult struct {
	Artists      []TemplateArtist `json:"artists"`
	TotalResults int              `json:"total_results"`

	// Pre-formatted display fields
	ResultCountText string `json:"result_count_text"` // "Found 5 artists" or "Found 1 artist"
}

// --- Template Data Formatters ---
// These functions convert raw data models into template-specific structs with pre-formatted display domain.

// toTitleCase converts a string to title case, replacing hyphens with spaces.
// This replaces the deprecated strings.Title function.
func toTitleCase(s string) string {
	s = strings.ReplaceAll(s, "-", " ")
	words := strings.Fields(s)
	for i, word := range words {
		if len(word) > 0 {
			words[i] = strings.ToUpper(string(word[0])) + strings.ToLower(word[1:])
		}
	}
	return strings.Join(words, " ")
}

// FormatArtistForTemplate converts a domain.Artist into a TemplateArtist with pre-formatted display fields.
func FormatArtistForTemplate(artist domain.Artist) TemplateArtist {
	memberCount := len(artist.Members)
	memberCountText := fmt.Sprintf("%d member", memberCount)
	if memberCount != 1 {
		memberCountText += "s"
	}

	concertCountText := fmt.Sprintf("%d concert", artist.ConcertCount)
	if artist.ConcertCount != 1 {
		concertCountText += "s"
	}

	return TemplateArtist{
		Name:             artist.Name,
		Slug:             artist.Slug,
		Image:            artist.Image,
		CreationYear:     artist.CreationYear,
		FirstAlbum:       artist.FirstAlbum,
		Members:          artist.Members,
		ConcertCount:     artist.ConcertCount,
		Countries:        artist.Countries,
		DisplayName:      artist.Name,
		MemberCountText:  memberCountText,
		CountriesText:    strings.Join(artist.Countries, ", "),
		ConcertCountText: concertCountText,
		MembersText:      strings.Join(artist.Members, ", "),
		CreationText:     fmt.Sprintf("Founded %d", artist.CreationYear),
	}
}

// FormatArtistsForTemplate converts a slice of domain.Artist into TemplateArtist structs.
func FormatArtistsForTemplate(artists []domain.Artist) []TemplateArtist {
	templateArtists := make([]TemplateArtist, len(artists))
	for i, artist := range artists {
		templateArtists[i] = FormatArtistForTemplate(artist)
	}
	return templateArtists
}

// FormatLocationForTemplate converts a domain.Location into a TemplateLocation with pre-formatted display fields.
func FormatLocationForTemplate(location domain.Location) TemplateLocation {
	artistCountText := fmt.Sprintf("%d artist", location.ArtistCount)
	if location.ArtistCount != 1 {
		artistCountText += "s"
	}

	concertCountText := fmt.Sprintf("%d concert", location.TotalConcerts)
	if location.TotalConcerts != 1 {
		concertCountText += "s"
	}

	yearRangeText := fmt.Sprintf("%d", location.EarliestYear)
	if location.EarliestYear != location.LatestYear {
		yearRangeText = fmt.Sprintf("%d - %d", location.EarliestYear, location.LatestYear)
	}

	return TemplateLocation{
		Name:             location.Name,
		Slug:             location.Slug,
		ArtistCount:      location.ArtistCount,
		TotalConcerts:    location.TotalConcerts,
		EarliestYear:     location.EarliestYear,
		LatestYear:       location.LatestYear,
		DisplayName:      toTitleCase(location.Name),
		ArtistCountText:  artistCountText,
		ConcertCountText: concertCountText,
		YearRangeText:    yearRangeText,
	}
}

// FormatFilterOptionsForTemplate converts domain.ArtistFilterOptions into TemplateFilterOptions with pre-formatted display.
func FormatFilterOptionsForTemplate(options domain.ArtistFilterOptions) TemplateFilterOptions {
	memberCountOptions := make([]string, len(options.MemberCounts))
	for i, count := range options.MemberCounts {
		if count == 1 {
			memberCountOptions[i] = "1 member"
		} else {
			memberCountOptions[i] = fmt.Sprintf("%d members", count)
		}
	}

	return TemplateFilterOptions{
		CreationYearMin:     options.CreationYearMin,
		CreationYearMax:     options.CreationYearMax,
		FirstAlbumYearMin:   options.FirstAlbumYearMin,
		FirstAlbumYearMax:   options.FirstAlbumYearMax,
		MemberCounts:        options.MemberCounts,
		Countries:           options.Countries,
		CreationYearRange:   fmt.Sprintf("%d - %d", options.CreationYearMin, options.CreationYearMax),
		FirstAlbumYearRange: fmt.Sprintf("%d - %d", options.FirstAlbumYearMin, options.FirstAlbumYearMax),
		MemberCountOptions:  memberCountOptions,
	}
}

// FormatAppliedFiltersForTemplate converts domain.ArtistFilterParams into TemplateAppliedFilters with pre-formatted display.
func FormatAppliedFiltersForTemplate(filters domain.ArtistFilterParams) TemplateAppliedFilters {
	template := TemplateAppliedFilters{
		CreationYearFrom:   filters.CreationYearFrom,
		CreationYearTo:     filters.CreationYearTo,
		FirstAlbumYearFrom: filters.FirstAlbumYearFrom,
		FirstAlbumYearTo:   filters.FirstAlbumYearTo,
		MemberCounts:       filters.MemberCounts,
		Countries:          filters.Countries,
	}

	// Format creation year text
	if filters.CreationYearFrom != nil && filters.CreationYearTo != nil {
		template.CreationYearText = fmt.Sprintf("%d - %d", *filters.CreationYearFrom, *filters.CreationYearTo)
	} else if filters.CreationYearFrom != nil {
		template.CreationYearText = fmt.Sprintf("From %d", *filters.CreationYearFrom)
	} else if filters.CreationYearTo != nil {
		template.CreationYearText = fmt.Sprintf("Up to %d", *filters.CreationYearTo)
	}

	// Format first album year text
	if filters.FirstAlbumYearFrom != nil && filters.FirstAlbumYearTo != nil {
		template.FirstAlbumYearText = fmt.Sprintf("%d - %d", *filters.FirstAlbumYearFrom, *filters.FirstAlbumYearTo)
	} else if filters.FirstAlbumYearFrom != nil {
		template.FirstAlbumYearText = fmt.Sprintf("From %d", *filters.FirstAlbumYearFrom)
	} else if filters.FirstAlbumYearTo != nil {
		template.FirstAlbumYearText = fmt.Sprintf("Up to %d", *filters.FirstAlbumYearTo)
	}

	// Format member counts text
	if len(filters.MemberCounts) > 0 {
		countStrs := make([]string, len(filters.MemberCounts))
		for i, count := range filters.MemberCounts {
			countStrs[i] = fmt.Sprintf("%d", count)
		}
		template.MemberCountsText = strings.Join(countStrs, ", ") + " members"
	}

	// Format countries text
	if len(filters.Countries) > 0 {
		template.CountriesText = strings.Join(filters.Countries, ", ")
	}

	// Check if any filters are active
	template.IsActive = filters.CreationYearFrom != nil || filters.CreationYearTo != nil ||
		filters.FirstAlbumYearFrom != nil || filters.FirstAlbumYearTo != nil ||
		len(filters.MemberCounts) > 0 || len(filters.Countries) > 0

	return template
}

// FormatSearchResultForTemplate converts domain.SearchResult into TemplateSearchResult with pre-formatted display.
func FormatSearchResultForTemplate(result domain.SearchResult) TemplateSearchResult {
	resultCountText := fmt.Sprintf("Found %d artist", result.TotalResults)
	if result.TotalResults != 1 {
		resultCountText += "s"
	}

	return TemplateSearchResult{
		Artists:         FormatArtistsForTemplate(result.Artists),
		TotalResults:    result.TotalResults,
		ResultCountText: resultCountText,
	}
}
