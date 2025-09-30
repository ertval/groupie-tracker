package data

import (
	"regexp"
	"strings"
)

var slugPattern = regexp.MustCompile(`[^a-z0-9]+`)

func slugify(input string) string {
	lower := strings.ToLower(strings.TrimSpace(input))
	lower = slugPattern.ReplaceAllString(lower, "-")
	return strings.Trim(lower, "-")
}

// Slugify is the exported helper that produces predictable URL-friendly slugs.
func Slugify(input string) string {
	return slugify(input)
}

// CountryFromLocation extracts a normalized country from the API location slug.
func CountryFromLocation(location string) string {
	parts := strings.Split(strings.ToLower(location), "-")
	if len(parts) == 0 {
		return ""
	}

	candidate := strings.TrimSpace(parts[len(parts)-1])
	switch candidate {
	case "usa", "us":
		return "USA"
	case "uk":
		return "UK"
	case "uae":
		return "UAE"
	}

	words := strings.Fields(strings.ReplaceAll(candidate, "-", " "))
	for i, word := range words {
		if word == "" {
			continue
		}
		words[i] = strings.ToUpper(word[:1]) + strings.ToLower(word[1:])
	}

	return strings.Join(words, " ")
}

// YearFromDate extracts the calendar year from a variety of date strings.
func YearFromDate(date string) int {
	if len(date) < 4 {
		return 0
	}

	if len(date) >= 10 && date[2] == '-' && date[5] == '-' {
		if year := parseYear(date[6:10]); year > 0 {
			return year
		}
	}

	return parseYear(date[:4])
}

// YearFromAlbum extracts the year component from the "first album" API field.
func YearFromAlbum(album string) int {
	return YearFromDate(album)
}

func parseYear(value string) int {
	if len(value) != 4 {
		return 0
	}

	year := 0
	for _, c := range value {
		if c < '0' || c > '9' {
			return 0
		}
		year = year*10 + int(c-'0')
	}

	if year <= 1900 || year >= 3000 {
		return 0
	}

	return year
}

func containsString(haystack []string, needle string) bool {
	for _, candidate := range haystack {
		if candidate == needle {
			return true
		}
	}
	return false
}
