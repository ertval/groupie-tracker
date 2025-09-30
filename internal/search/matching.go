package search

import (
	"strconv"

	"groupie-tracker/internal/data"
)

func matchesQuery(artist data.Artist, normalizedQuery string) bool {
	if normalizedQuery == "" {
		return true
	}

	if stringsContains(normalize(artist.Name), normalizedQuery) {
		return true
	}

	for _, member := range artist.Members {
		if stringsContains(normalize(member), normalizedQuery) {
			return true
		}
	}

	for _, concert := range artist.Concerts {
		if stringsContains(normalize(concert.Location), normalizedQuery) {
			return true
		}
		if stringsContains(normalize(concert.Country), normalizedQuery) {
			return true
		}
	}

	if stringsContains(strconv.Itoa(artist.CreationYear), normalizedQuery) {
		return true
	}

	if stringsContains(normalize(artist.FirstAlbum), normalizedQuery) {
		return true
	}

	return false
}
