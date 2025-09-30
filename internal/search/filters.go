package search

import (
	"sort"
	"strings"

	"groupie-tracker/internal/data"
)

// IsEmpty reports whether any filter has been activated.
func (f Filters) IsEmpty() bool {
	return f.CreationYearMin == 0 &&
		f.CreationYearMax == 0 &&
		f.FirstAlbumYearMin == 0 &&
		f.FirstAlbumYearMax == 0 &&
		len(f.MemberCounts) == 0 &&
		len(f.Countries) == 0
}

func filterArtists(artists []data.Artist, filters Filters) []data.Artist {
	if len(artists) == 0 || filters.IsEmpty() {
		return artists
	}

	filtered := make([]data.Artist, 0, len(artists))
	for _, artist := range artists {
		if matchesFilters(artist, filters) {
			filtered = append(filtered, artist)
		}
	}

	return filtered
}

func matchesFilters(artist data.Artist, filters Filters) bool {
	if filters.CreationYearMin > 0 && artist.CreationYear < filters.CreationYearMin {
		return false
	}
	if filters.CreationYearMax > 0 && artist.CreationYear > filters.CreationYearMax {
		return false
	}

	albumYear := data.YearFromAlbum(artist.FirstAlbum)
	if filters.FirstAlbumYearMin > 0 && albumYear < filters.FirstAlbumYearMin {
		return false
	}
	if filters.FirstAlbumYearMax > 0 && albumYear > filters.FirstAlbumYearMax {
		return false
	}

	if len(filters.MemberCounts) > 0 {
		memberCount := len(artist.Members)
		match := false
		for _, allowed := range filters.MemberCounts {
			if memberCount == allowed {
				match = true
				break
			}
		}
		if !match {
			return false
		}
	}

	if len(filters.Countries) > 0 {
		match := false
		for _, filterCountry := range filters.Countries {
			for _, artistCountry := range artist.Countries {
				if strings.EqualFold(artistCountry, filterCountry) {
					match = true
					break
				}
			}
			if match {
				break
			}
		}
		if !match {
			return false
		}
	}

	return true
}

func computeFilterOptions(artists []data.Artist) FilterOptions {
	if len(artists) == 0 {
		return FilterOptions{}
	}

	options := FilterOptions{
		CreationYearMin:   artists[0].CreationYear,
		CreationYearMax:   artists[0].CreationYear,
		FirstAlbumYearMin: 9999,
	}

	memberCounts := make(map[int]struct{})
	countrySet := make(map[string]struct{})

	for _, artist := range artists {
		if year := artist.CreationYear; year > 0 {
			if year < options.CreationYearMin {
				options.CreationYearMin = year
			}
			if year > options.CreationYearMax {
				options.CreationYearMax = year
			}
		}

		if albumYear := data.YearFromAlbum(artist.FirstAlbum); albumYear > 0 {
			if albumYear < options.FirstAlbumYearMin {
				options.FirstAlbumYearMin = albumYear
			}
			if albumYear > options.FirstAlbumYearMax {
				options.FirstAlbumYearMax = albumYear
			}
		}

		if count := len(artist.Members); count > 0 {
			memberCounts[count] = struct{}{}
		}

		for _, country := range artist.Countries {
			if country == "" {
				continue
			}
			countrySet[country] = struct{}{}
		}
	}

	for count := range memberCounts {
		options.MemberCounts = append(options.MemberCounts, count)
	}
	sort.Ints(options.MemberCounts)

	for country := range countrySet {
		options.Countries = append(options.Countries, country)
	}
	sort.Strings(options.Countries)

	if options.FirstAlbumYearMin == 9999 {
		options.FirstAlbumYearMin = options.CreationYearMin
	}

	return options
}
