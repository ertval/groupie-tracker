package search

import "strings"

func normalize(input string) string {
	return strings.ToLower(strings.TrimSpace(input))
}

func stringsContains(haystack, needle string) bool {
	if needle == "" {
		return true
	}
	return strings.Contains(haystack, needle)
}
