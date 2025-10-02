package data

import (
	"strconv"
	"strings"
)

// extractCountryFromLocation normalizes a location string and returns a display-ready country name.
func extractCountryFromLocation(location string) string {
	parts := strings.Split(strings.ToLower(strings.TrimSpace(location)), "-")
	if len(parts) == 0 {
		return ""
	}

	country := strings.TrimSpace(parts[len(parts)-1])
	if country == "" {
		return ""
	}

	switch country {
	case "usa", "us":
		return "USA"
	case "uk":
		return "UK"
	case "uae":
		return "UAE"
	}

	words := strings.Fields(strings.ReplaceAll(country, "-", " "))
	for i, word := range words {
		if len(word) == 0 {
			continue
		}
		words[i] = strings.ToUpper(word[:1]) + strings.ToLower(word[1:])
	}
	return strings.Join(words, " ")
}

// extractYearFromDate attempts to parse common date formats and return the year component.
func extractYearFromDate(dateStr string) int {
	dateStr = strings.TrimSpace(dateStr)
	if len(dateStr) < 4 {
		return 0
	}

	if len(dateStr) >= 10 && dateStr[2] == '-' && dateStr[5] == '-' {
		if year, err := strconv.Atoi(dateStr[6:10]); err == nil {
			return year
		}
	}

	if year, err := strconv.Atoi(dateStr[:4]); err == nil && year > 1900 && year < 3000 {
		return year
	}

	return 0
}
