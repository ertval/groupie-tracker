package search

import (
	"net/url"
	"strconv"
	"strings"
)

// ParseFilters builds a Filters struct from HTTP form/query values.
func ParseFilters(values url.Values) Filters {
	filters := Filters{}

	filters.CreationYearMin = positiveInt(values.Get("creation_year_min"))
	filters.CreationYearMax = positiveInt(values.Get("creation_year_max"))
	filters.FirstAlbumYearMin = positiveInt(values.Get("first_album_year_min"))
	filters.FirstAlbumYearMax = positiveInt(values.Get("first_album_year_max"))

	if memberValues, ok := values["member_counts"]; ok {
		for _, v := range memberValues {
			if count := positiveInt(v); count > 0 {
				filters.MemberCounts = append(filters.MemberCounts, count)
			}
		}
	}

	if countryValues, ok := values["countries"]; ok {
		for _, v := range countryValues {
			if trimmed := strings.TrimSpace(v); trimmed != "" {
				filters.Countries = append(filters.Countries, trimmed)
			}
		}
	}

	return filters
}

func positiveInt(value string) int {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0
	}
	n, err := strconv.Atoi(value)
	if err != nil || n < 0 {
		return 0
	}
	return n
}
